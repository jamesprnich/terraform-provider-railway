package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceDataSource_byId(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, mockFixtures{
		"getService": `{"data":{"service":{"id":"00000000-0000-0000-0000-000000000003","name":"api-server","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
data "railway_service" "test" {
  id = "00000000-0000-0000-0000-000000000003"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_service.test", "id", "00000000-0000-0000-0000-000000000003"),
					resource.TestCheckResourceAttr("data.railway_service.test", "name", "api-server"),
					resource.TestCheckResourceAttr("data.railway_service.test", "project_id", "00000000-0000-0000-0000-000000000001"),
				),
			},
		},
	})
}

func TestServiceDataSource_byName(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, mockFixtures{
		"getProjectServices": `{"data":{"project":{"services":{"edges":[{"node":{"id":"00000000-0000-0000-0000-000000000003","name":"api-server"}}]}}}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
data "railway_service" "test" {
  name       = "api-server"
  project_id = "00000000-0000-0000-0000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_service.test", "id", "00000000-0000-0000-0000-000000000003"),
					resource.TestCheckResourceAttr("data.railway_service.test", "name", "api-server"),
					resource.TestCheckResourceAttr("data.railway_service.test", "project_id", "00000000-0000-0000-0000-000000000001"),
				),
			},
		},
	})
}
