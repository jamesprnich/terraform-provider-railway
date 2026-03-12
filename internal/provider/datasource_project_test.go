package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProjectDataSource_byId(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, mockFixtures{
		"getProject": `{"data":{"project":{"id":"00000000-0000-0000-0000-000000000001","name":"test-project","description":"A test project","isPublic":false,"prDeploys":false,"workspace":null,"environments":{"edges":[]}}}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
data "railway_project" "test" {
  id = "00000000-0000-0000-0000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_project.test", "id", "00000000-0000-0000-0000-000000000001"),
					resource.TestCheckResourceAttr("data.railway_project.test", "name", "test-project"),
					resource.TestCheckResourceAttr("data.railway_project.test", "description", "A test project"),
				),
			},
		},
	})
}

func TestProjectDataSource_byName(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, mockFixtures{
		"listProjects": `{"data":{"projects":{"edges":[{"node":{"id":"00000000-0000-0000-0000-000000000001","name":"test-project","description":"A test project"}}]}}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
data "railway_project" "test" {
  name = "test-project"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_project.test", "id", "00000000-0000-0000-0000-000000000001"),
					resource.TestCheckResourceAttr("data.railway_project.test", "name", "test-project"),
					resource.TestCheckResourceAttr("data.railway_project.test", "description", "A test project"),
				),
			},
		},
	})
}
