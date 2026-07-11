package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
