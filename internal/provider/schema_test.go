package provider

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	fwprovider_schema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestAllResourcesHaveSchemas(t *testing.T) {
	t.Parallel()

	resources := []struct {
		name    string
		factory func() fwresource.Resource
	}{
		{"railway_project", NewProjectResource},
		{"railway_environment", NewEnvironmentResource},
		{"railway_service", NewServiceResource},
		{"railway_service_instance", NewServiceInstanceResource},
		{"railway_variable", NewVariableResource},
		{"railway_variable_collection", NewVariableCollectionResource},
		{"railway_shared_variable", NewSharedVariableResource},
		{"railway_volume", NewVolumeResource},
		{"railway_volume_backup_schedule", NewVolumeBackupScheduleResource},
		{"railway_service_domain", NewServiceDomainResource},
		{"railway_custom_domain", NewCustomDomainResource},
		{"railway_tcp_proxy", NewTcpProxyResource},
		{"railway_deployment_trigger", NewDeploymentTriggerResource},
		{"railway_egress_gateway", NewEgressGatewayResource},
		{"railway_private_network", NewPrivateNetworkResource},
		{"railway_private_network_endpoint", NewPrivateNetworkEndpointResource},
		{"railway_project_token", NewProjectTokenResource},
		{"railway_trusted_domain", NewTrustedDomainResource},
		{"railway_notification_rule", NewNotificationRuleResource},
		{"railway_bucket", NewBucketResource},
		{"railway_ssh_public_key", NewSshPublicKeyResource},
		{"railway_project_member", NewProjectMemberResource},
	}

	for _, tc := range resources {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			schemaResp := &fwresource.SchemaResponse{}
			tc.factory().Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema has errors: %v", schemaResp.Diagnostics.Errors())
			}

			if len(schemaResp.Schema.Attributes) == 0 {
				t.Fatal("schema has no attributes")
			}
		})
	}
}

func TestAllDataSourcesHaveSchemas(t *testing.T) {
	t.Parallel()

	dataSources := []struct {
		name    string
		factory func() fwdatasource.DataSource
	}{
		{"railway_project", NewProjectDataSource},
		{"railway_environment", NewEnvironmentDataSource},
		{"railway_service", NewServiceDataSource},
	}

	for _, tc := range dataSources {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			schemaResp := &fwdatasource.SchemaResponse{}
			tc.factory().Schema(ctx, fwdatasource.SchemaRequest{}, schemaResp)

			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema has errors: %v", schemaResp.Diagnostics.Errors())
			}

			if len(schemaResp.Schema.Attributes) == 0 {
				t.Fatal("schema has no attributes")
			}
		})
	}
}

func TestEgressGatewayResourceSchema_immutableFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	NewEgressGatewayResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	for _, name := range []string{"service_id", "environment_id"} {
		attr, ok := schemaResp.Schema.Attributes[name].(fwschema.StringAttribute)
		if !ok {
			t.Errorf("attribute %q not found or wrong type", name)
			continue
		}
		if len(attr.PlanModifiers) == 0 {
			t.Errorf("attribute %q should have plan modifiers (RequiresReplace)", name)
		}
	}
}

// TestServiceResourceSchema_v0_11_shape verifies railway_service was correctly
// reduced to a shell in v0.11.0. Fields that used to trigger env-less
// mutations (source_*, cron_schedule, regions, etc.) are gone.
func TestServiceResourceSchema_v0_11_shape(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	NewServiceResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	expected := map[string]bool{
		"id":             true,
		"name":           true,
		"project_id":     true,
		"environment_id": true,
		"icon":           true,
		"volume":         true,
	}
	for k := range schemaResp.Schema.Attributes {
		if !expected[k] {
			t.Errorf("railway_service has unexpected attribute %q — should have been stripped in v0.11.0", k)
		}
	}
	for k := range expected {
		if _, ok := schemaResp.Schema.Attributes[k]; !ok {
			t.Errorf("railway_service missing expected attribute %q", k)
		}
	}

	envIdAttr, ok := schemaResp.Schema.Attributes["environment_id"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("environment_id is not a StringAttribute")
	}
	if len(envIdAttr.PlanModifiers) == 0 {
		t.Error("environment_id should have RequiresReplace + UseStateForUnknown plan modifiers")
	}
}

// TestEnvironmentResourceSchema_forkAttribute verifies source_environment_id
// exists on railway_environment and is create-only (RequiresReplace).
func TestEnvironmentResourceSchema_forkAttribute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	NewEnvironmentResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	srcEnvAttr, ok := schemaResp.Schema.Attributes["source_environment_id"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("source_environment_id not found or wrong type")
	}
	if len(srcEnvAttr.PlanModifiers) == 0 {
		t.Error("source_environment_id should have RequiresReplace + UseStateForUnknown plan modifiers")
	}
	if !srcEnvAttr.Optional {
		t.Error("source_environment_id should be Optional at the schema level (strict-mode check runs in Create)")
	}
}

// TestProviderSchema_strictEnvScoping verifies the provider block accepts the
// strict_env_scoping opt-out flag.
func TestProviderSchema_strictEnvScoping(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaResp := &fwprovider.SchemaResponse{}
	New("test")().Schema(ctx, fwprovider.SchemaRequest{}, schemaResp)

	attr, ok := schemaResp.Schema.Attributes["strict_env_scoping"].(fwprovider_schema.BoolAttribute)
	if !ok {
		t.Fatal("strict_env_scoping not found or wrong type")
	}
	if !attr.Optional {
		t.Error("strict_env_scoping must be Optional (default = true resolved in Configure)")
	}
}
