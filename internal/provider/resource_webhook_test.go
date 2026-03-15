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

func TestAccWebhookResourceDefault(t *testing.T) {
	t.Skip("Webhook API types (WebhookCreateInput, ProjectWebhook) are not in Railway's public GraphQL schema")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebhookDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccWebhookResourceConfig("https://example.com/webhook-test", `["deploy.completed"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_webhook.test", "id"),
					resource.TestCheckResourceAttr("railway_webhook.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_webhook.test", "url", "https://example.com/webhook-test"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.#", "1"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.0", "deploy.completed"),
				),
			},
			// Import
			{
				ResourceName: "railway_webhook.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["railway_webhook.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return testAccProjectId + ":" + rs.Primary.ID, nil
				},
				ImportStateVerify: true,
			},
			// Update URL and filters
			{
				Config: testAccWebhookResourceConfig("https://example.com/webhook-updated", `["deploy.completed", "deploy.started"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_webhook.test", "id"),
					resource.TestCheckResourceAttr("railway_webhook.test", "url", "https://example.com/webhook-updated"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.#", "2"),
				),
			},
		},
	})
}

func TestAccWebhookResource_disappears(t *testing.T) {
	t.Skip("Webhook API types (WebhookCreateInput, ProjectWebhook) are not in Railway's public GraphQL schema")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWebhookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookResourceConfig("https://example.com/webhook-disappears", `["deploy.completed"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_webhook.test", "id"),
					testAccCheckWebhookDisappears("railway_webhook.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccWebhookResourceConfig(url, filters string) string {
	return fmt.Sprintf(`
resource "railway_webhook" "test" {
  project_id = "%s"
  url        = "%s"
  filters    = %s
}
`, testAccProjectId, url, filters)
}

func TestWebhookResource_basic(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	server := newMockGraphQLServer(t, mockFixtures{
		"createWebhook": `{"data":{"webhookCreate":{"id":"wh-123","url":"https://example.com/hook","projectId":"` + projectId + `","filters":["deploy.completed"],"lastStatus":0}}}`,
		"getWebhooks":   `{"data":{"webhooks":{"edges":[{"node":{"id":"wh-123","url":"https://example.com/hook","projectId":"` + projectId + `","filters":["deploy.completed"],"lastStatus":0}}]}}}`,
		"deleteWebhook": `{"data":{"webhookDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_webhook" "test" {
  project_id = "` + projectId + `"
  url        = "https://example.com/hook"
  filters    = ["deploy.completed"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_webhook.test", "id", "wh-123"),
					resource.TestCheckResourceAttr("railway_webhook.test", "project_id", projectId),
					resource.TestCheckResourceAttr("railway_webhook.test", "url", "https://example.com/hook"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.#", "1"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.0", "deploy.completed"),
				),
			},
		},
	})
}

func TestWebhookResource_update(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"

	v1Response := `{"data":{"webhooks":{"edges":[{"node":{"id":"wh-456","url":"https://example.com/hook-v1","projectId":"` + projectId + `","filters":["deploy.completed"],"lastStatus":0}}]}}}`
	v2Response := `{"data":{"webhooks":{"edges":[{"node":{"id":"wh-456","url":"https://example.com/hook-v2","projectId":"` + projectId + `","filters":["deploy.completed","deploy.started"],"lastStatus":0}}]}}}`

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

		switch req.OperationName {
		case "createWebhook":
			fmt.Fprint(w, `{"data":{"webhookCreate":{"id":"wh-456","url":"https://example.com/hook-v1","projectId":"`+projectId+`","filters":["deploy.completed"],"lastStatus":0}}}`)
		case "updateWebhook":
			updated = true
			fmt.Fprint(w, `{"data":{"webhookUpdate":{"id":"wh-456","url":"https://example.com/hook-v2","projectId":"`+projectId+`","filters":["deploy.completed","deploy.started"],"lastStatus":0}}}`)
		case "getWebhooks":
			if updated {
				fmt.Fprint(w, v2Response)
			} else {
				fmt.Fprint(w, v1Response)
			}
		case "deleteWebhook":
			fmt.Fprint(w, `{"data":{"webhookDelete":true}}`)
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
resource "railway_webhook" "test" {
  project_id = "` + projectId + `"
  url        = "https://example.com/hook-v1"
  filters    = ["deploy.completed"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_webhook.test", "id", "wh-456"),
					resource.TestCheckResourceAttr("railway_webhook.test", "url", "https://example.com/hook-v1"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.#", "1"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.0", "deploy.completed"),
				),
			},
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_webhook" "test" {
  project_id = "` + projectId + `"
  url        = "https://example.com/hook-v2"
  filters    = ["deploy.completed", "deploy.started"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_webhook.test", "id", "wh-456"),
					resource.TestCheckResourceAttr("railway_webhook.test", "url", "https://example.com/hook-v2"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.#", "2"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.0", "deploy.completed"),
					resource.TestCheckResourceAttr("railway_webhook.test", "filters.1", "deploy.started"),
				),
			},
		},
	})
}

func TestWebhookResource_noFilters(t *testing.T) {
	projectId := "99999999-8888-7777-6666-555555555555"

	server := newMockGraphQLServer(t, mockFixtures{
		"createWebhook": `{"data":{"webhookCreate":{"id":"wh-789","url":"https://example.com/no-filters","projectId":"` + projectId + `","filters":[],"lastStatus":0}}}`,
		"getWebhooks":   `{"data":{"webhooks":{"edges":[{"node":{"id":"wh-789","url":"https://example.com/no-filters","projectId":"` + projectId + `","filters":[],"lastStatus":0}}]}}}`,
		"deleteWebhook": `{"data":{"webhookDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_webhook" "test" {
  project_id = "` + projectId + `"
  url        = "https://example.com/no-filters"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_webhook.test", "id", "wh-789"),
					resource.TestCheckResourceAttr("railway_webhook.test", "url", "https://example.com/no-filters"),
					resource.TestCheckResourceAttr("railway_webhook.test", "project_id", projectId),
				),
			},
		},
	})
}

func TestWebhookResource_disappears(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createWebhook": `{"data":{"webhookCreate":{"id":"wh-dis","url":"https://example.com/hook","projectId":"` + projectId + `","filters":["deploy.completed"],"lastStatus":0}}}`,
		"getWebhooks":   `{"data":{"webhooks":{"edges":[{"node":{"id":"wh-dis","url":"https://example.com/hook","projectId":"` + projectId + `","filters":["deploy.completed"],"lastStatus":0}}]}}}`,
		"deleteWebhook": `{"data":{"webhookDelete":true}}`,
	}, "getWebhooks")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_webhook" "test" {
  project_id = "` + projectId + `"
  url        = "https://example.com/hook"
  filters    = ["deploy.completed"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_webhook.test", "id", "wh-dis"),
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

func TestWebhookResource_import(t *testing.T) {
	projectId := "abcdefab-1234-5678-9abc-def012345678"

	server := newMockGraphQLServer(t, mockFixtures{
		"createWebhook": `{"data":{"webhookCreate":{"id":"wh-imp","url":"https://example.com/import","projectId":"` + projectId + `","filters":["deploy.completed"],"lastStatus":0}}}`,
		"getWebhooks":   `{"data":{"webhooks":{"edges":[{"node":{"id":"wh-imp","url":"https://example.com/import","projectId":"` + projectId + `","filters":["deploy.completed"],"lastStatus":0}}]}}}`,
		"deleteWebhook": `{"data":{"webhookDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_webhook" "test" {
  project_id = "` + projectId + `"
  url        = "https://example.com/import"
  filters    = ["deploy.completed"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_webhook.test", "id", "wh-imp"),
				),
			},
			{
				ResourceName:      "railway_webhook.test",
				ImportState:       true,
				ImportStateId:     projectId + ":wh-imp",
				ImportStateVerify: true,
			},
		},
	})
}
