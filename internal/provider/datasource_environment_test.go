package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentDataSource_byId(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, mockFixtures{
		"getEnvironment": `{"data":{"environment":{"id":"00000000-0000-0000-0000-000000000002","name":"production","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
data "railway_environment" "test" {
  id = "00000000-0000-0000-0000-000000000002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_environment.test", "id", "00000000-0000-0000-0000-000000000002"),
					resource.TestCheckResourceAttr("data.railway_environment.test", "name", "production"),
					resource.TestCheckResourceAttr("data.railway_environment.test", "project_id", "00000000-0000-0000-0000-000000000001"),
				),
			},
		},
	})
}

func TestEnvironmentDataSource_byName(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, mockFixtures{
		"getEnvironments": `{"data":{"environments":{"edges":[{"node":{"id":"00000000-0000-0000-0000-000000000002","name":"production","projectId":"00000000-0000-0000-0000-000000000001"}}]}}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
data "railway_environment" "test" {
  name       = "production"
  project_id = "00000000-0000-0000-0000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_environment.test", "id", "00000000-0000-0000-0000-000000000002"),
					resource.TestCheckResourceAttr("data.railway_environment.test", "name", "production"),
					resource.TestCheckResourceAttr("data.railway_environment.test", "project_id", "00000000-0000-0000-0000-000000000001"),
				),
			},
		},
	})
}
