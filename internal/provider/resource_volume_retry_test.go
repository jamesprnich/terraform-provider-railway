package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// =============================================================================
// Volume retry unit tests
//
// These prove the §4.3 post-create Read retry actually recovers from
// Railway's eventual-consistency window. Live acceptance tests can only
// observe the retry firing by luck of the window — a mock server that
// deterministically fails N times then succeeds proves both directions of
// the retry contract:
//
//   - the retry recovers from transient "not found" (positive case)
//   - the retry does NOT mask a persistent "not found" (negative case)
// =============================================================================

// newFlakyVolumeReadMockServer returns a mock GraphQL server that fails the
// first `failReads` invocations of `getVolumeInstances` with a "not found"
// error, then serves the normal fixtures. Every other operation uses the
// standard fixture map.
func newFlakyVolumeReadMockServer(t *testing.T, fixtures mockFixtures, failReads int32) *httptest.Server {
	t.Helper()

	var readCount int32

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock: failed to decode request: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if req.OperationName == "getVolumeInstances" {
			n := atomic.AddInt32(&readCount, 1)
			if n <= failReads {
				_, _ = fmt.Fprint(w, `{"errors":[{"message":"volume instance not found (transient)"}]}`)
				return
			}
		}

		response, ok := fixtures[req.OperationName]
		if !ok {
			t.Errorf("mock: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, `{"errors":[{"message":"unexpected operation: %s"}]}`, req.OperationName)
			return
		}
		_, _ = fmt.Fprint(w, response)
	}))
}

// TestVolumeResource_retryRecoversFromTransientNotFound proves the positive
// case: Railway takes a few seconds to make the volume queryable after
// volumeCreate returns; the provider retries the read and the create
// succeeds without an operator-visible error.
func TestVolumeResource_retryRecoversFromTransientNotFound(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	volumeCreateResp := fmt.Sprintf(`{"data":{"volumeCreate":{"id":"vol-123","name":"my-volume","projectId":"%s"}}}`, projectId)
	volumeInstancesResp := fmt.Sprintf(`{"data":{"project":{"volumes":{"edges":[{"node":{"id":"vol-123","name":"my-volume","volumeInstances":{"edges":[{"node":{"id":"vi-123","environmentId":"%s","serviceId":"%s","mountPath":"/data","sizeMB":1024}}]}}}]}}}}`, envId, serviceId)

	// Fail the first 3 getVolumeInstances calls (simulating a ~few-second
	// eventual-consistency window), then serve the fixture.
	srv := newFlakyVolumeReadMockServer(t, mockFixtures{
		"createVolume":       volumeCreateResp,
		"getVolumeInstances": volumeInstancesResp,
		"deleteVolume":       `{"data":{"volumeDelete":true}}`,
	}, 3)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/data"
}
`, projectId, serviceId, envId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "id", "vol-123"),
					resource.TestCheckResourceAttr("railway_volume.test", "volume_instance_id", "vi-123"),
					resource.TestCheckResourceAttr("railway_volume.test", "size_mb", "1024"),
				),
			},
		},
	})
}

// TestVolumeResource_retryDoesNotMaskPersistentNotFound proves the negative
// case: if the volume genuinely never becomes readable (something is really
// wrong), the retry loop bounds out at 30s and the operator sees a clear
// error rather than the create hanging forever.
func TestVolumeResource_retryDoesNotMaskPersistentNotFound(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	volumeCreateResp := fmt.Sprintf(`{"data":{"volumeCreate":{"id":"vol-123","name":"my-volume","projectId":"%s"}}}`, projectId)

	// Fail every getVolumeInstances call — no recovery possible.
	// The retry budget (30s) will be exhausted and the create errors.
	srv := newFlakyVolumeReadMockServer(t, mockFixtures{
		"createVolume":       volumeCreateResp,
		"getVolumeInstances": `{"data":{"project":{"volumes":{"edges":[]}}}}`, // wouldn't matter — never reached
		"deleteVolume":       `{"data":{"volumeDelete":true}}`,
	}, 9999)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/data"
}
`, projectId, serviceId, envId),
				ExpectError: regexp.MustCompile(`Unable to read volume after creation`),
			},
		},
	})
}

// =============================================================================
// Volume rename retry unit tests
//
// These prove the retryUpdateContext wrapping added around the post-create
// rename (Create) and the rename/mount-path calls (Update) actually closes
// the taint/prevent_destroy deadlock: a transient rename failure recovers
// automatically instead of tainting the volume, and — for the case where
// retries are genuinely exhausted — the volume is left as a visible,
// replaceable taint rather than a silent, untracked orphan.
// =============================================================================

// newFlakyOperationsMockServer returns a mock GraphQL server that fails the
// first N invocations of each named operation in failCounts with a transient
// error, then serves the normal fixtures for that operation. Operations not
// listed in failCounts always use the standard fixture map.
func newFlakyOperationsMockServer(t *testing.T, fixtures mockFixtures, failCounts map[string]int32) *httptest.Server {
	t.Helper()

	counts := make(map[string]*int32, len(failCounts))
	for op := range failCounts {
		counts[op] = new(int32)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock: failed to decode request: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if limit, ok := failCounts[req.OperationName]; ok {
			n := atomic.AddInt32(counts[req.OperationName], 1)
			if n <= limit {
				_, _ = fmt.Fprint(w, `{"errors":[{"message":"Problem processing request (transient)"}]}`)
				return
			}
		}

		response, ok := fixtures[req.OperationName]
		if !ok {
			t.Errorf("mock: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, `{"errors":[{"message":"unexpected operation: %s"}]}`, req.OperationName)
			return
		}
		_, _ = fmt.Fprint(w, response)
	}))
}

// TestVolumeResource_createRenameRetryRecoversFromTransientFailure proves the
// positive case for the Create-path fix: a transient error on the post-create
// rename (updateVolume) recovers via retryUpdateContext, and Create succeeds
// with the correct final name — no operator-visible error, no taint.
func TestVolumeResource_createRenameRetryRecoversFromTransientFailure(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	volumeCreateResp := fmt.Sprintf(`{"data":{"volumeCreate":{"id":"vol-rename-ok","name":"Volume","projectId":"%s"}}}`, projectId)
	volumeInstancesResp := fmt.Sprintf(`{"data":{"project":{"volumes":{"edges":[{"node":{"id":"vol-rename-ok","name":"postgres-data","volumeInstances":{"edges":[{"node":{"id":"vi-rename-ok","environmentId":"%s","serviceId":"%s","mountPath":"/data","sizeMB":1024}}]}}}]}}}}`, envId, serviceId)

	srv := newFlakyOperationsMockServer(t, mockFixtures{
		"createVolume":       volumeCreateResp,
		"updateVolume":       `{"data":{"volumeUpdate":{"id":"vol-rename-ok","name":"postgres-data"}}}`,
		"getVolumeInstances": volumeInstancesResp,
		"deleteVolume":       `{"data":{"volumeDelete":true}}`,
	}, map[string]int32{"updateVolume": 3})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/data"
  name           = "postgres-data"
}
`, projectId, serviceId, envId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "id", "vol-rename-ok"),
					resource.TestCheckResourceAttr("railway_volume.test", "name", "postgres-data"),
				),
			},
		},
	})
}

// newTaintThenReplaceMockServer returns a stateful mock GraphQL server that
// tracks volume id -> name across calls, like a tiny in-memory Railway. The
// first volume it ever creates ("vol-taint-1") can never be successfully
// renamed — every updateVolume call against it fails transiently, simulating
// Railway's rename never actually committing server-side. Any later volume
// renames normally. Used to prove that a Create which exhausts its rename
// retries leaves a visible, replaceable taint in state (not a silent,
// untracked orphan): the next apply must see the doomed volume, destroy it,
// create a fresh one, and succeed cleanly.
func newTaintThenReplaceMockServer(t *testing.T, projectId, envId, serviceId, mountPath string) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	var createCount int32
	volumes := map[string]string{} // id -> current name

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock: failed to decode request: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch req.OperationName {
		case "createVolume":
			n := atomic.AddInt32(&createCount, 1)
			id := fmt.Sprintf("vol-taint-%d", n)
			volumes[id] = "Volume"
			_, _ = fmt.Fprintf(w, `{"data":{"volumeCreate":{"id":"%s","name":"Volume","projectId":"%s"}}}`, id, projectId)
		case "updateVolume":
			var vars struct {
				Id    string `json:"id"`
				Input struct {
					Name string `json:"name"`
				} `json:"input"`
			}
			_ = json.Unmarshal(req.Variables, &vars)
			if vars.Id == "vol-taint-1" {
				// The doomed first volume: rename never succeeds, no matter how
				// many times it's retried.
				_, _ = fmt.Fprint(w, `{"errors":[{"message":"Problem processing request (transient)"}]}`)
				return
			}
			volumes[vars.Id] = vars.Input.Name
			_, _ = fmt.Fprintf(w, `{"data":{"volumeUpdate":{"id":"%s","name":"%s"}}}`, vars.Id, vars.Input.Name)
		case "getVolumeInstances":
			edges := make([]string, 0, len(volumes))
			for id, name := range volumes {
				edges = append(edges, fmt.Sprintf(`{"node":{"id":"%s","name":"%s","volumeInstances":{"edges":[{"node":{"id":"vi-%s","environmentId":"%s","serviceId":"%s","mountPath":"%s","sizeMB":1024}}]}}}`, id, name, id, envId, serviceId, mountPath))
			}
			_, _ = fmt.Fprintf(w, `{"data":{"project":{"volumes":{"edges":[%s]}}}}`, strings.Join(edges, ","))
		case "deleteVolume":
			var vars struct {
				Id string `json:"id"`
			}
			_ = json.Unmarshal(req.Variables, &vars)
			delete(volumes, vars.Id)
			_, _ = fmt.Fprint(w, `{"data":{"volumeDelete":true}}`)
		default:
			t.Errorf("mock: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
}

// TestVolumeResource_createRenameRetryExhausted_leavesVisibleTaintNotOrphan
// proves the negative case: when the rename genuinely never succeeds (retries
// exhausted), Create must surface a clear error AND the volume it actually
// created must remain visible in state as a replaceable taint. A plan showing
// "Create" instead of "Replace" on the next step would mean Create stopped
// persisting state on rename failure — the exact invisible-orphan regression
// this test exists to catch (a real, billed Railway volume with no Terraform
// record of it at all, worse than a visible taint).
func TestVolumeResource_createRenameRetryExhausted_leavesVisibleTaintNotOrphan(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	srv := newTaintThenReplaceMockServer(t, projectId, envId, serviceId, "/data")
	defer srv.Close()

	config := testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/data"
  name           = "postgres-data"
}
`, projectId, serviceId, envId)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Unable to update volume name`),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("railway_volume.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "id", "vol-taint-2"),
					resource.TestCheckResourceAttr("railway_volume.test", "name", "postgres-data"),
				),
			},
		},
	})
}

// newFlakyVolumeLifecycleMockServer returns a stateful mock GraphQL server
// for a single volume across its whole lifecycle: create, then rename/mount
// path updates. It tracks the volume's current name and mount path, mutating
// them only when a mutation actually succeeds — updateVolume and
// updateVolumeInstance calls fail the first N times per failCounts with a
// transient error before succeeding. getVolumeInstances always reflects the
// current tracked values, so a refresh before a change lands sees the OLD
// values and the post-mutation readback sees the NEW ones. A static fixture
// would leak the end-state into the pre-change refresh and hide the resource
// change from the plan (Terraform would see config == already-refreshed
// state and report no-op) — this must stay stateful to test the actual
// update path.
func newFlakyVolumeLifecycleMockServer(t *testing.T, id, envId, serviceId string, failCounts map[string]int32) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	var name, mountPath string

	counts := make(map[string]*int32, len(failCounts))
	for op := range failCounts {
		counts[op] = new(int32)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock: failed to decode request: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		if limit, ok := failCounts[req.OperationName]; ok {
			n := atomic.AddInt32(counts[req.OperationName], 1)
			if n <= limit {
				_, _ = fmt.Fprint(w, `{"errors":[{"message":"Problem processing request (transient)"}]}`)
				return
			}
		}

		switch req.OperationName {
		case "createVolume":
			var vars struct {
				Input struct {
					MountPath string `json:"mountPath"`
				} `json:"input"`
			}
			_ = json.Unmarshal(req.Variables, &vars)
			name = "Volume"
			mountPath = vars.Input.MountPath
			_, _ = fmt.Fprintf(w, `{"data":{"volumeCreate":{"id":"%s","name":"%s"}}}`, id, name)
		case "updateVolume":
			var vars struct {
				Input struct {
					Name string `json:"name"`
				} `json:"input"`
			}
			_ = json.Unmarshal(req.Variables, &vars)
			name = vars.Input.Name
			_, _ = fmt.Fprintf(w, `{"data":{"volumeUpdate":{"id":"%s","name":"%s"}}}`, id, name)
		case "updateVolumeInstance":
			var vars struct {
				Input struct {
					MountPath string `json:"mountPath"`
				} `json:"input"`
			}
			_ = json.Unmarshal(req.Variables, &vars)
			mountPath = vars.Input.MountPath
			_, _ = fmt.Fprint(w, `{"data":{"volumeInstanceUpdate":true}}`)
		case "getVolumeInstances":
			_, _ = fmt.Fprintf(w, `{"data":{"project":{"volumes":{"edges":[{"node":{"id":"%s","name":"%s","volumeInstances":{"edges":[{"node":{"id":"vi-%s","environmentId":"%s","serviceId":"%s","mountPath":"%s","sizeMB":1024}}]}}}]}}}}`, id, name, id, envId, serviceId, mountPath)
		case "deleteVolume":
			_, _ = fmt.Fprint(w, `{"data":{"volumeDelete":true}}`)
		default:
			t.Errorf("mock: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
}

// TestVolumeResource_updateRetryRenamesInPlaceUnderPreventDestroy proves the
// Update-path fix and the prevent_destroy safety property together: renaming
// AND moving the mount path on an existing, prevent_destroy-protected volume
// both recover from a couple of transient failures via retryUpdateContext and
// apply as in-place updates. If the plan had instead proposed replacing the
// resource, this apply would fail outright — OpenTofu refuses to destroy a
// prevent_destroy resource — so a clean, successful apply here is itself
// direct proof the plan action was Update, not Replace. The final step drops
// prevent_destroy so the test framework's own post-test cleanup can tear the
// resource down; it asserts nothing itself.
func TestVolumeResource_updateRetryRenamesInPlaceUnderPreventDestroy(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	srv := newFlakyVolumeLifecycleMockServer(t, "vol-pd", envId, serviceId, map[string]int32{"updateVolume": 2, "updateVolumeInstance": 2})
	defer srv.Close()

	resourceBlock := func(mountPath, nameAttr, lifecycleBlock string) string {
		return testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "%s"
%s
%s
}
`, projectId, serviceId, envId, mountPath, nameAttr, lifecycleBlock)
	}

	preventDestroy := `
  lifecycle {
    prevent_destroy = true
  }`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// Initial create with no name set (server default) — keeps the
				// rename retry budget untouched for the next step's assertion.
				Config: resourceBlock("/data", "", preventDestroy),
			},
			{
				Config: resourceBlock("/var/lib/postgresql/data", `  name           = "postgres-data"`, preventDestroy),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("railway_volume.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "id", "vol-pd"),
					resource.TestCheckResourceAttr("railway_volume.test", "name", "postgres-data"),
					resource.TestCheckResourceAttr("railway_volume.test", "mount_path", "/var/lib/postgresql/data"),
				),
			},
			{
				// Drop prevent_destroy so the framework's post-test destroy sweep
				// can clean up. No new assertions — this step is teardown-only.
				Config: resourceBlock("/var/lib/postgresql/data", `  name           = "postgres-data"`, ""),
			},
		},
	})
}
