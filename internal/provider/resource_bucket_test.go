package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance test — runs within the per-run fixture project. The bucket is project-scoped
// and is cleaned up by cascade when the fixture project is deleted. Note: Railway has no
// bucketDelete API, so the resource's Delete is a state-only no-op; the bucket itself
// is removed when the parent project is destroyed at the end of the run.
func TestAccBucketResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "railway_bucket" "test" {
  name       = "acc-bucket"
  project_id = "%s"
}
`, testAccProjectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_bucket.test", "id"),
					resource.TestCheckResourceAttr("railway_bucket.test", "name", "acc-bucket"),
					resource.TestCheckResourceAttr("railway_bucket.test", "project_id", testAccProjectId),
				),
			},
			// Update name in place (calls bucketUpdate API)
			{
				Config: fmt.Sprintf(`
resource "railway_bucket" "test" {
  name       = "acc-bucket-renamed"
  project_id = "%s"
}
`, testAccProjectId),
				Check: resource.TestCheckResourceAttr("railway_bucket.test", "name", "acc-bucket-renamed"),
			},
		},
	})
}

func TestBucketResource_basic(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"

	server := newMockGraphQLServer(t, mockFixtures{
		"createBucket":      `{"data":{"bucketCreate":{"id":"bkt-1","name":"data","projectId":"` + projectId + `","createdAt":"2026-05-15T00:00:00Z","updatedAt":"2026-05-15T00:00:00Z"}}}`,
		"getProjectBuckets": `{"data":{"project":{"buckets":{"edges":[{"node":{"id":"bkt-1","name":"data","projectId":"` + projectId + `","createdAt":"2026-05-15T00:00:00Z","updatedAt":"2026-05-15T00:00:00Z"}}]}}}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_bucket" "test" {
  name       = "data"
  project_id = "` + projectId + `"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_bucket.test", "id", "bkt-1"),
					resource.TestCheckResourceAttr("railway_bucket.test", "name", "data"),
					resource.TestCheckResourceAttr("railway_bucket.test", "project_id", projectId),
				),
			},
		},
	})
}

func TestBucketResource_update(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"

	var mu sync.Mutex
	updated := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock server: failed to decode request body: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		name := "data-v1"
		if updated {
			name = "data-v2"
		}
		switch req.OperationName {
		case "createBucket":
			fmt.Fprint(w, `{"data":{"bucketCreate":{"id":"bkt-2","name":"data-v1","projectId":"`+projectId+`","createdAt":"2026-05-15T00:00:00Z","updatedAt":"2026-05-15T00:00:00Z"}}}`)
		case "updateBucket":
			updated = true
			fmt.Fprint(w, `{"data":{"bucketUpdate":{"id":"bkt-2","name":"data-v2","projectId":"`+projectId+`","createdAt":"2026-05-15T00:00:00Z","updatedAt":"2026-05-15T00:00:00Z"}}}`)
		case "getProjectBuckets":
			fmt.Fprintf(w, `{"data":{"project":{"buckets":{"edges":[{"node":{"id":"bkt-2","name":"%s","projectId":"%s","createdAt":"2026-05-15T00:00:00Z","updatedAt":"2026-05-15T00:00:00Z"}}]}}}}`, name, projectId)
		default:
			t.Errorf("mock server: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_bucket" "test" {
  name       = "data-v1"
  project_id = "` + projectId + `"
}`,
				Check: resource.TestCheckResourceAttr("railway_bucket.test", "name", "data-v1"),
			},
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_bucket" "test" {
  name       = "data-v2"
  project_id = "` + projectId + `"
}`,
				Check: resource.TestCheckResourceAttr("railway_bucket.test", "name", "data-v2"),
			},
		},
	})
}

func TestBucketResource_disappears(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createBucket":      `{"data":{"bucketCreate":{"id":"bkt-3","name":"data","projectId":"` + projectId + `","createdAt":"2026-05-15T00:00:00Z","updatedAt":"2026-05-15T00:00:00Z"}}}`,
		"getProjectBuckets": `{"data":{"project":{"buckets":{"edges":[{"node":{"id":"bkt-3","name":"data","projectId":"` + projectId + `","createdAt":"2026-05-15T00:00:00Z","updatedAt":"2026-05-15T00:00:00Z"}}]}}}}`,
	}, "getProjectBuckets")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_bucket" "test" {
  name       = "data"
  project_id = "` + projectId + `"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_bucket.test", "id", "bkt-3"),
					func(s *terraform.State) error {
						disappear()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
