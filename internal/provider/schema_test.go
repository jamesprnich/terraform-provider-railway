package provider

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
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
