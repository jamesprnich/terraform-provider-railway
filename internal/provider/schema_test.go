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
		{"railway_webhook", NewWebhookResource},
		{"railway_egress_gateway", NewEgressGatewayResource},
		{"railway_private_network", NewPrivateNetworkResource},
		{"railway_private_network_endpoint", NewPrivateNetworkEndpointResource},
		{"railway_deployment_trigger", NewDeploymentTriggerResource},
		{"railway_volume_backup_schedule", NewVolumeBackupScheduleResource},
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

func TestWebhookResourceSchema_attributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	NewWebhookResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	attrs := schemaResp.Schema.Attributes

	// Check required attributes
	for _, name := range []string{"project_id", "url"} {
		attr, ok := attrs[name]
		if !ok {
			t.Errorf("attribute %q not found", name)
			continue
		}
		strAttr, ok := attr.(fwschema.StringAttribute)
		if !ok {
			t.Errorf("attribute %q is not StringAttribute", name)
			continue
		}
		if !strAttr.Required {
			t.Errorf("attribute %q should be Required", name)
		}
	}

	// Check computed attributes
	idAttr, ok := attrs["id"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("id attribute not found or wrong type")
	}
	if !idAttr.Computed {
		t.Error("id should be Computed")
	}

	// Check filters is Optional list
	filtersAttr, ok := attrs["filters"].(fwschema.ListAttribute)
	if !ok {
		t.Fatal("filters attribute not found or wrong type")
	}
	if !filtersAttr.Optional {
		t.Error("filters should be Optional")
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
