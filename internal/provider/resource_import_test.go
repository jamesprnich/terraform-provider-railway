package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var importFormatRegex = regexp.MustCompile("Expected import identifier with format")

// TestEnvironmentResource_importInvalid tests that malformed import IDs
// are rejected by the railway_environment resource.
// Expected format: project_id:name (2 parts).
func TestEnvironmentResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_environment" "test" {
  name       = "test"
  project_id = "11111111-2222-3333-4444-555555555555"
}
`,
				ResourceName:  "railway_environment.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_environment" "test" {
  name       = "test"
  project_id = "11111111-2222-3333-4444-555555555555"
}
`,
				ResourceName:  "railway_environment.test",
				ImportState:   true,
				ImportStateId: "abc:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_environment" "test" {
  name       = "test"
  project_id = "11111111-2222-3333-4444-555555555555"
}
`,
				ResourceName:  "railway_environment.test",
				ImportState:   true,
				ImportStateId: ":name",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestServiceInstanceResource_importInvalid tests that malformed import IDs
// are rejected by the railway_service_instance resource.
// Expected format: service_id:environment_id (2 parts).
func TestServiceInstanceResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_service_instance.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_service_instance.test",
				ImportState:   true,
				ImportStateId: "abc:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_service_instance.test",
				ImportState:   true,
				ImportStateId: ":env-id",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestVariableResource_importInvalid tests that malformed import IDs
// are rejected by the railway_variable resource.
// Expected format: service_id:environment_name:name (3 parts).
func TestVariableResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_variable" "test" {
  name           = "TEST_VAR"
  value          = "test"
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_variable.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_variable" "test" {
  name           = "TEST_VAR"
  value          = "test"
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_variable.test",
				ImportState:   true,
				ImportStateId: "svc:env:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_variable" "test" {
  name           = "TEST_VAR"
  value          = "test"
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_variable.test",
				ImportState:   true,
				ImportStateId: ":env:name",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestVariableCollectionResource_importInvalid tests that malformed import IDs
// are rejected by the railway_variable_collection resource.
// Expected format: service_id:environment_name:name1:name2:... (3+ parts).
func TestVariableCollectionResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_variable_collection" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
  variables = [
    { name = "VAR1", value = "val1" },
  ]
}
`,
				ResourceName:  "railway_variable_collection.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_variable_collection" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
  variables = [
    { name = "VAR1", value = "val1" },
  ]
}
`,
				ResourceName:  "railway_variable_collection.test",
				ImportState:   true,
				ImportStateId: "svc:env",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_variable_collection" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
  variables = [
    { name = "VAR1", value = "val1" },
  ]
}
`,
				ResourceName:  "railway_variable_collection.test",
				ImportState:   true,
				ImportStateId: "svc::name1",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestSharedVariableResource_importInvalid tests that malformed import IDs
// are rejected by the railway_shared_variable resource.
// Expected format: project_id:environment_name:name (3 parts).
func TestSharedVariableResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_shared_variable" "test" {
  name           = "SHARED_VAR"
  value          = "test"
  project_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_shared_variable.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_shared_variable" "test" {
  name           = "SHARED_VAR"
  value          = "test"
  project_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_shared_variable.test",
				ImportState:   true,
				ImportStateId: "proj:env:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_shared_variable" "test" {
  name           = "SHARED_VAR"
  value          = "test"
  project_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_shared_variable.test",
				ImportState:   true,
				ImportStateId: ":env:name",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestVolumeResource_importInvalid tests that malformed import IDs
// are rejected by the railway_volume resource.
// Expected format: project_id:volume_id:service_id:environment_id (4 parts).
func TestVolumeResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_volume" "test" {
  project_id     = "11111111-2222-3333-4444-555555555555"
  service_id     = "22222222-3333-4444-5555-666666666666"
  environment_id = "33333333-4444-5555-6666-777777777777"
  mount_path     = "/data"
}
`,
				ResourceName:  "railway_volume.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_volume" "test" {
  project_id     = "11111111-2222-3333-4444-555555555555"
  service_id     = "22222222-3333-4444-5555-666666666666"
  environment_id = "33333333-4444-5555-6666-777777777777"
  mount_path     = "/data"
}
`,
				ResourceName:  "railway_volume.test",
				ImportState:   true,
				ImportStateId: "a:b:c:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_volume" "test" {
  project_id     = "11111111-2222-3333-4444-555555555555"
  service_id     = "22222222-3333-4444-5555-666666666666"
  environment_id = "33333333-4444-5555-6666-777777777777"
  mount_path     = "/data"
}
`,
				ResourceName:  "railway_volume.test",
				ImportState:   true,
				ImportStateId: ":vol:svc:env",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestServiceDomainResource_importInvalid tests that malformed import IDs
// are rejected by the railway_service_domain resource.
// Expected format: service_id:environment_name:domain (3 parts).
func TestServiceDomainResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_domain" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_service_domain.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_domain" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_service_domain.test",
				ImportState:   true,
				ImportStateId: "svc:env:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_domain" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_service_domain.test",
				ImportState:   true,
				ImportStateId: ":env:domain",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestCustomDomainResource_importInvalid tests that malformed import IDs
// are rejected by the railway_custom_domain resource.
// Expected format: service_id:environment_name:domain (3 parts).
func TestCustomDomainResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "example.com"
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_custom_domain.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "example.com"
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_custom_domain.test",
				ImportState:   true,
				ImportStateId: "svc:env:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "example.com"
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_custom_domain.test",
				ImportState:   true,
				ImportStateId: ":env:domain",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestTcpProxyResource_importInvalid tests that malformed import IDs
// are rejected by the railway_tcp_proxy resource.
// Expected format: service_id:environment_id:tcp_proxy_id (3 parts).
func TestTcpProxyResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_tcp_proxy" "test" {
  application_port = 5432
  service_id       = "11111111-2222-3333-4444-555555555555"
  environment_id   = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_tcp_proxy.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_tcp_proxy" "test" {
  application_port = 5432
  service_id       = "11111111-2222-3333-4444-555555555555"
  environment_id   = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_tcp_proxy.test",
				ImportState:   true,
				ImportStateId: "svc:env:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_tcp_proxy" "test" {
  application_port = 5432
  service_id       = "11111111-2222-3333-4444-555555555555"
  environment_id   = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_tcp_proxy.test",
				ImportState:   true,
				ImportStateId: ":env:proxy",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestWebhookResource_importInvalid tests that malformed import IDs
// are rejected by the railway_webhook resource.
// Expected format: project_id:webhook_id (2 parts).
func TestWebhookResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_webhook" "test" {
  project_id = "11111111-2222-3333-4444-555555555555"
  url        = "https://example.com/webhook"
}
`,
				ResourceName:  "railway_webhook.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_webhook" "test" {
  project_id = "11111111-2222-3333-4444-555555555555"
  url        = "https://example.com/webhook"
}
`,
				ResourceName:  "railway_webhook.test",
				ImportState:   true,
				ImportStateId: "proj:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_webhook" "test" {
  project_id = "11111111-2222-3333-4444-555555555555"
  url        = "https://example.com/webhook"
}
`,
				ResourceName:  "railway_webhook.test",
				ImportState:   true,
				ImportStateId: ":webhook-id",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestDeploymentTriggerResource_importInvalid tests that malformed import IDs
// are rejected by the railway_deployment_trigger resource.
// Expected format: project_id:environment_id:service_id:trigger_id (4 parts).
func TestDeploymentTriggerResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_deployment_trigger" "test" {
  project_id      = "11111111-2222-3333-4444-555555555555"
  environment_id  = "22222222-3333-4444-5555-666666666666"
  service_id      = "33333333-4444-5555-6666-777777777777"
  repository      = "owner/repo"
  branch          = "main"
  source_provider = "github"
}
`,
				ResourceName:  "railway_deployment_trigger.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_deployment_trigger" "test" {
  project_id      = "11111111-2222-3333-4444-555555555555"
  environment_id  = "22222222-3333-4444-5555-666666666666"
  service_id      = "33333333-4444-5555-6666-777777777777"
  repository      = "owner/repo"
  branch          = "main"
  source_provider = "github"
}
`,
				ResourceName:  "railway_deployment_trigger.test",
				ImportState:   true,
				ImportStateId: "a:b:c:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_deployment_trigger" "test" {
  project_id      = "11111111-2222-3333-4444-555555555555"
  environment_id  = "22222222-3333-4444-5555-666666666666"
  service_id      = "33333333-4444-5555-6666-777777777777"
  repository      = "owner/repo"
  branch          = "main"
  source_provider = "github"
}
`,
				ResourceName:  "railway_deployment_trigger.test",
				ImportState:   true,
				ImportStateId: ":env:svc:trigger",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestEgressGatewayResource_importInvalid tests that malformed import IDs
// are rejected by the railway_egress_gateway resource.
// Expected format: service_id:environment_id (2 parts).
func TestEgressGatewayResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_egress_gateway.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_egress_gateway.test",
				ImportState:   true,
				ImportStateId: "svc:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
}
`,
				ResourceName:  "railway_egress_gateway.test",
				ImportState:   true,
				ImportStateId: ":env-id",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestPrivateNetworkResource_importInvalid tests that malformed import IDs
// are rejected by the railway_private_network resource.
// Expected format: environment_id:network_public_id (2 parts).
func TestPrivateNetworkResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network" "test" {
  project_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
  name           = "test-network"
}
`,
				ResourceName:  "railway_private_network.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network" "test" {
  project_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
  name           = "test-network"
}
`,
				ResourceName:  "railway_private_network.test",
				ImportState:   true,
				ImportStateId: "env:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network" "test" {
  project_id     = "11111111-2222-3333-4444-555555555555"
  environment_id = "22222222-3333-4444-5555-666666666666"
  name           = "test-network"
}
`,
				ResourceName:  "railway_private_network.test",
				ImportState:   true,
				ImportStateId: ":network-id",
				ExpectError:   importFormatRegex,
			},
		},
	})
}

// TestPrivateNetworkEndpointResource_importInvalid tests that malformed import IDs
// are rejected by the railway_private_network_endpoint resource.
// Expected format: environment_id:private_network_id:service_id (3 parts).
func TestPrivateNetworkEndpointResource_importInvalid(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "net-123"
  service_id         = "11111111-2222-3333-4444-555555555555"
  environment_id     = "22222222-3333-4444-5555-666666666666"
  service_name       = "test-service"
}
`,
				ResourceName:  "railway_private_network_endpoint.test",
				ImportState:   true,
				ImportStateId: "only-one-part",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "net-123"
  service_id         = "11111111-2222-3333-4444-555555555555"
  environment_id     = "22222222-3333-4444-5555-666666666666"
  service_name       = "test-service"
}
`,
				ResourceName:  "railway_private_network_endpoint.test",
				ImportState:   true,
				ImportStateId: "env:net:",
				ExpectError:   importFormatRegex,
			},
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "net-123"
  service_id         = "11111111-2222-3333-4444-555555555555"
  environment_id     = "22222222-3333-4444-5555-666666666666"
  service_name       = "test-service"
}
`,
				ResourceName:  "railway_private_network_endpoint.test",
				ImportState:   true,
				ImportStateId: ":net:svc",
				ExpectError:   importFormatRegex,
			},
		},
	})
}
