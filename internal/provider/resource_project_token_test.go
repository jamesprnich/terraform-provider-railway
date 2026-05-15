package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance test — runs within the per-run fixture project. The token is project-scoped
// and is cleaned up automatically when the fixture project is deleted at the end of the run.
func TestAccProjectTokenResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "railway_project_token" "test" {
  name           = "acc-token"
  project_id     = "%s"
  environment_id = "%s"
}
`, testAccProjectId, testAccEnvironmentId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_project_token.test", "id"),
					resource.TestCheckResourceAttr("railway_project_token.test", "name", "acc-token"),
					resource.TestCheckResourceAttr("railway_project_token.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_project_token.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttrSet("railway_project_token.test", "token"),
				),
			},
		},
	})
}

func TestProjectTokenResource_basic(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"
	envId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	server := newMockGraphQLServer(t, mockFixtures{
		"createProjectToken": `{"data":{"projectTokenCreate":"raw-token-secret"}}`,
		"getProjectTokens":   `{"data":{"projectTokens":{"edges":[{"node":{"id":"tok-1","name":"ci","projectId":"` + projectId + `","environmentId":"` + envId + `"}}]}}}`,
		"deleteProjectToken": `{"data":{"projectTokenDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_project_token" "test" {
  name           = "ci"
  project_id     = "` + projectId + `"
  environment_id = "` + envId + `"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_project_token.test", "id", "tok-1"),
					resource.TestCheckResourceAttr("railway_project_token.test", "name", "ci"),
					resource.TestCheckResourceAttr("railway_project_token.test", "project_id", projectId),
					resource.TestCheckResourceAttr("railway_project_token.test", "environment_id", envId),
					resource.TestCheckResourceAttr("railway_project_token.test", "token", "raw-token-secret"),
				),
			},
		},
	})
}

func TestProjectTokenResource_disappears(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"
	envId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createProjectToken": `{"data":{"projectTokenCreate":"raw-token-secret"}}`,
		"getProjectTokens":   `{"data":{"projectTokens":{"edges":[{"node":{"id":"tok-2","name":"ci","projectId":"` + projectId + `","environmentId":"` + envId + `"}}]}}}`,
		"deleteProjectToken": `{"data":{"projectTokenDelete":true}}`,
	}, "getProjectTokens")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_project_token" "test" {
  name           = "ci"
  project_id     = "` + projectId + `"
  environment_id = "` + envId + `"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_project_token.test", "id", "tok-2"),
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
