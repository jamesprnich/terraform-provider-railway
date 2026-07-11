package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// Cooldown-retry unit tests
//
// Railway enforces per-workspace and per-user cooldowns on creates:
//
//   projectCreate     — "1 project per 30 seconds"
//   environmentCreate — "Only one environment can be created per user every 30s"
//
// These tests prove the provider recovers transparently — no test-side
// time.Sleep needed. They also prove the retry is bounded: a genuinely
// broken call still surfaces its error.
// =============================================================================

func TestIsCreationCooldown_true(t *testing.T) {
	cases := []string{
		"You are creating projects too quickly. This workspace allows 1 project per 30 seconds. Try again shortly.",
		"projectCreate You are creating projects too quickly.",
		"only one environment can be created per user every 30s",
		"Whoa there pal! Only one environment can be created per user every 30s. Try again in a sec",
	}
	for _, msg := range cases {
		t.Run(msg[:20], func(t *testing.T) {
			if !isCreationCooldown(errors.New(msg)) {
				t.Errorf("expected isCreationCooldown to be true for %q", msg)
			}
		})
	}
}

func TestIsCreationCooldown_false(t *testing.T) {
	cases := []string{
		"",
		"some other error",
		"project not found",
		"unauthorized",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			var err error
			if msg != "" {
				err = errors.New(msg)
			}
			if isCreationCooldown(err) {
				t.Errorf("expected isCreationCooldown to be false for %q", msg)
			}
		})
	}
}

// TestRetryOnCooldownContext_recoversFromCooldown proves the retry recovers
// from a project-cooldown-shaped error and eventually returns success.
func TestRetryOnCooldownContext_recoversFromCooldown(t *testing.T) {
	var attempts int32
	err := retryOnCooldownContext(context.Background(), 30*time.Second, func() error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return errors.New("You are creating projects too quickly. This workspace allows 1 project per 30 seconds")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected recovery, got: %s", err)
	}
	if attempts < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", attempts)
	}
}

// TestRetryOnCooldownContext_bailsOnNonRetryable proves non-cooldown errors
// surface immediately without retrying.
func TestRetryOnCooldownContext_bailsOnNonRetryable(t *testing.T) {
	var attempts int32
	err := retryOnCooldownContext(context.Background(), 5*time.Second, func() error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("some genuinely broken thing")
	})
	if err == nil {
		t.Fatal("expected non-retryable error to surface, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt, got %d", attempts)
	}
}

// newCooldownMockServer serves the given fixture but fails the first
// `failCreates` invocations of `createProject` (or `createEnvironment`) with
// the exact cooldown error message Railway returns.
func newCooldownMockServer(t *testing.T, fixtures mockFixtures, opName string, failCreates int32, cooldownMsg string) *httptest.Server {
	t.Helper()
	var count int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock: decode failed: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		if req.OperationName == opName {
			n := atomic.AddInt32(&count, 1)
			if n <= failCreates {
				_, _ = fmt.Fprint(w, `{"errors":[{"message":"`+cooldownMsg+`"}]}`)
				return
			}
		}

		response, ok := fixtures[req.OperationName]
		if !ok {
			t.Errorf("mock: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprint(w, response)
	}))
}

// TestProjectResource_recoversFromProjectCooldown proves the provider-side
// retry lets a project apply succeed even when Railway rejects the first two
// createProject calls with the "1 per 30 seconds" cooldown.
func TestProjectResource_recoversFromProjectCooldown(t *testing.T) {
	pid := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	// Fixture must include workspace (Project fragment reads Workspace)
	// and prDeploys/isPublic which the resource maps into Terraform state.
	projectJSON := `{"id":"` + pid + `","name":"test-proj","description":"","isPublic":false,"prDeploys":false,"workspace":{"id":"11111111-1111-1111-1111-111111111111"},"environments":{"edges":[{"node":{"id":"env-default","name":"production","projectId":"` + pid + `"}}]}}`
	createResp := `{"data":{"projectCreate":` + projectJSON + `}}`
	getResp := `{"data":{"project":` + projectJSON + `}}`

	srv := newCooldownMockServer(t, mockFixtures{
		"createProject": createResp,
		"getProject":    getResp,
		"deleteProject": `{"data":{"projectDelete":true}}`,
	}, "createProject", 2,
		"You are creating projects too quickly. This workspace allows 1 project per 30 seconds. Try again shortly.")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_project" "test" {
  name = "test-proj"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_project.test", "id", pid),
					resource.TestCheckResourceAttr("railway_project.test", "name", "test-proj"),
				),
			},
		},
	})
}

// TestEnvironmentResource_recoversFromEnvCooldown proves the same for
// environmentCreate.
func TestEnvironmentResource_recoversFromEnvCooldown(t *testing.T) {
	envId := "99999999-0000-1111-2222-333333333333"
	projId := "00000000-0000-0000-0000-000000000001"
	srcId := "00000000-0000-0000-0000-000000000007"

	createResp := `{"data":{"environmentCreate":{"id":"` + envId + `","name":"dev","projectId":"` + projId + `","isEphemeral":false,"sourceEnvironment":{"id":"` + srcId + `"}}}}`
	getResp := `{"data":{"environments":{"edges":[{"node":{"id":"` + envId + `","name":"dev","projectId":"` + projId + `","isEphemeral":false,"sourceEnvironment":{"id":"` + srcId + `"}}}]}}}`

	srv := newCooldownMockServer(t, mockFixtures{
		"createEnvironment": createResp,
		"getEnvironments":   getResp,
		"deleteEnvironment": `{"data":{"environmentDelete":true}}`,
	}, "createEnvironment", 2,
		"Whoa there pal! Only one environment can be created per user every 30s. Try again in a sec")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_environment" "test" {
  name                  = "dev"
  project_id            = "` + projId + `"
  source_environment_id = "` + srcId + `"
}
`,
				Check: resource.TestCheckResourceAttr("railway_environment.test", "id", envId),
			},
		},
	})
}

// TestProjectResource_boundsOnPersistentCooldown proves the retry doesn't hang
// forever — if the cooldown keeps firing beyond the budget, the error surfaces.
// We use a short retryOnCooldownContext-shaped test via the direct helper.
func TestRetryOnCooldownContext_boundsOnPersistentCooldown(t *testing.T) {
	var attempts int32
	err := retryOnCooldownContext(context.Background(), 3*time.Second, func() error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("You are creating projects too quickly")
	})
	if err == nil {
		t.Fatal("expected persistent cooldown to eventually surface, got nil")
	}
	// Regex sanity — the underlying error message should propagate.
	if !regexp.MustCompile(`(?i)creating projects too quickly`).MatchString(err.Error()) {
		t.Errorf("expected propagated error to reference the cooldown message, got: %s", err)
	}
	if attempts < 2 {
		t.Errorf("expected multiple retry attempts, got %d", attempts)
	}
}
