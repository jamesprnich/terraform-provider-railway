package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvironmentResource_nameNotForceNew(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	NewEnvironmentResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	nameAttr := schemaResp.Schema.Attributes["name"]
	strAttr, ok := nameAttr.(fwschema.StringAttribute)
	if !ok {
		t.Fatal("name attribute is not a StringAttribute")
	}

	if len(strAttr.PlanModifiers) > 0 {
		t.Fatal("name attribute should not have any plan modifiers (RequiresReplace was removed)")
	}
}

func TestEnvironmentResource_createAndRead(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, mockFixtures{
		"createEnvironment": `{"data":{"environmentCreate":{"id":"00000000-0000-0000-0000-000000000099","name":"staging","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
		"getEnvironment":    `{"data":{"environment":{"id":"00000000-0000-0000-0000-000000000099","name":"staging","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
		"deleteEnvironment": `{"data":{"environmentDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_environment" "test" {
  name       = "staging"
  project_id = "00000000-0000-0000-0000-000000000001"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_environment.test", "id", "00000000-0000-0000-0000-000000000099"),
					resource.TestCheckResourceAttr("railway_environment.test", "name", "staging"),
					resource.TestCheckResourceAttr("railway_environment.test", "project_id", "00000000-0000-0000-0000-000000000001"),
				),
			},
		},
	})
}
