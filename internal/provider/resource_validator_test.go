package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// UUID regex validator tests — must be an id (UUID format)
// =============================================================================

func TestServiceResource_invalidProjectId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name       = "test-service"
  project_id = "not-a-uuid"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestEnvironmentResource_invalidProjectId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_environment" "test" {
  name       = "staging"
  project_id = "invalid"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestVariableResource_invalidEnvironmentId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_variable" "test" {
  name           = "MY_VAR"
  value          = "my-value"
  environment_id = "bad-env-id"
  service_id     = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestVariableResource_invalidServiceId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_variable" "test" {
  name           = "MY_VAR"
  value          = "my-value"
  environment_id = "00000000-0000-0000-0000-000000000001"
  service_id     = "not-uuid"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestSharedVariableResource_invalidProjectId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_shared_variable" "test" {
  name           = "SHARED_VAR"
  value          = "shared-value"
  environment_id = "00000000-0000-0000-0000-000000000001"
  project_id     = "not-a-uuid"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestVariableCollectionResource_invalidEnvironmentId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_variable_collection" "test" {
  environment_id = "bad-env-id"
  service_id     = "00000000-0000-0000-0000-000000000001"
  variables = [
    { name = "VAR1", value = "val1" }
  ]
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestVolumeResource_invalidProjectId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_volume" "test" {
  project_id     = "not-uuid"
  service_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  mount_path     = "/data"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestWebhookResource_invalidProjectId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_webhook" "test" {
  project_id = "bad-id"
  url        = "https://example.com/webhook"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestCustomDomainResource_invalidEnvironmentId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "app.example.com"
  environment_id = "bad-env"
  service_id     = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestTcpProxyResource_invalidEnvironmentId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_tcp_proxy" "test" {
  application_port = 5432
  environment_id   = "bad-env"
  service_id       = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestServiceDomainResource_invalidEnvironmentId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_domain" "test" {
  environment_id = "bad-env"
  service_id     = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestDeploymentTriggerResource_invalidServiceId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_deployment_trigger" "test" {
  service_id      = "bad-service-id"
  environment_id  = "00000000-0000-0000-0000-000000000001"
  project_id      = "00000000-0000-0000-0000-000000000002"
  repository      = "owner/repo"
  branch          = "main"
  source_provider = "github"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestEgressGatewayResource_invalidServiceId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "bad-id"
  environment_id = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestPrivateNetworkResource_invalidProjectId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network" "test" {
  project_id     = "bad-id"
  environment_id = "00000000-0000-0000-0000-000000000001"
  name           = "test-network"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestPrivateNetworkEndpointResource_invalidServiceId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "pn-123"
  service_id         = "bad-id"
  environment_id     = "00000000-0000-0000-0000-000000000001"
  service_name       = "my-service"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

func TestServiceInstanceResource_invalidServiceId(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "bad-id"
  environment_id = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`must be an id`),
			},
		},
	})
}

// =============================================================================
// URL format validator tests — webhook URL must be HTTP or HTTPS
// =============================================================================

func TestWebhookResource_invalidUrl(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_webhook" "test" {
  project_id = "00000000-0000-0000-0000-000000000001"
  url        = "ftp://example.com/webhook"
}`,
				ExpectError: regexp.MustCompile(`must be a valid HTTP or HTTPS URL`),
			},
		},
	})
}

func TestWebhookResource_invalidUrlNoScheme(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_webhook" "test" {
  project_id = "00000000-0000-0000-0000-000000000001"
  url        = "example.com/webhook"
}`,
				ExpectError: regexp.MustCompile(`must be a valid HTTP or HTTPS URL`),
			},
		},
	})
}

// =============================================================================
// String length validator tests — UTF8LengthAtLeast
// =============================================================================

func TestServiceResource_emptyName(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name       = ""
  project_id = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

func TestEnvironmentResource_emptyName(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_environment" "test" {
  name       = ""
  project_id = "00000000-0000-0000-0000-000000000001"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

func TestVariableResource_emptyName(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_variable" "test" {
  name           = ""
  value          = "some-value"
  environment_id = "00000000-0000-0000-0000-000000000001"
  service_id     = "00000000-0000-0000-0000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

func TestSharedVariableResource_emptyName(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_shared_variable" "test" {
  name           = ""
  value          = "some-value"
  environment_id = "00000000-0000-0000-0000-000000000001"
  project_id     = "00000000-0000-0000-0000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

func TestServiceResource_cronScheduleTooShort(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name          = "test-svc"
  project_id    = "00000000-0000-0000-0000-000000000001"
  cron_schedule = "short"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 9`),
			},
		},
	})
}

func TestVolumeResource_emptyMountPath(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_volume" "test" {
  project_id     = "00000000-0000-0000-0000-000000000001"
  service_id     = "00000000-0000-0000-0000-000000000002"
  environment_id = "00000000-0000-0000-0000-000000000003"
  mount_path     = ""
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

func TestCustomDomainResource_emptyDomain(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_custom_domain" "test" {
  domain         = ""
  environment_id = "00000000-0000-0000-0000-000000000001"
  service_id     = "00000000-0000-0000-0000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

func TestDeploymentTriggerResource_emptyRepository(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_deployment_trigger" "test" {
  service_id      = "00000000-0000-0000-0000-000000000001"
  environment_id  = "00000000-0000-0000-0000-000000000002"
  project_id      = "00000000-0000-0000-0000-000000000003"
  repository      = ""
  branch          = "main"
  source_provider = "github"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

func TestDeploymentTriggerResource_emptyBranch(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_deployment_trigger" "test" {
  service_id      = "00000000-0000-0000-0000-000000000001"
  environment_id  = "00000000-0000-0000-0000-000000000002"
  project_id      = "00000000-0000-0000-0000-000000000003"
  repository      = "owner/repo"
  branch          = ""
  source_provider = "github"
}`,
				ExpectError: regexp.MustCompile(`(?i)must be at least 1`),
			},
		},
	})
}

// =============================================================================
// Integer range validator tests — port ranges, replicas
// =============================================================================

func TestTcpProxyResource_portTooLow(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_tcp_proxy" "test" {
  application_port = 0
  environment_id   = "00000000-0000-0000-0000-000000000001"
  service_id       = "00000000-0000-0000-0000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`must be at least 1`),
			},
		},
	})
}

func TestTcpProxyResource_portTooHigh(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_tcp_proxy" "test" {
  application_port = 70000
  environment_id   = "00000000-0000-0000-0000-000000000001"
  service_id       = "00000000-0000-0000-0000-000000000002"
}`,
				ExpectError: regexp.MustCompile(`must be at most 65535`),
			},
		},
	})
}

func TestCustomDomainResource_targetPortTooLow(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "app.example.com"
  environment_id = "00000000-0000-0000-0000-000000000001"
  service_id     = "00000000-0000-0000-0000-000000000002"
  target_port    = 0
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 65535`),
			},
		},
	})
}

func TestCustomDomainResource_targetPortTooHigh(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "app.example.com"
  environment_id = "00000000-0000-0000-0000-000000000001"
  service_id     = "00000000-0000-0000-0000-000000000002"
  target_port    = 99999
}`,
				ExpectError: regexp.MustCompile(`must be between 1 and 65535`),
			},
		},
	})
}

func TestServiceResource_replicasTooLow(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name       = "test-svc"
  project_id = "00000000-0000-0000-0000-000000000001"
  regions = [
    {
      region       = "us-west2"
      num_replicas = 0
    }
  ]
}`,
				ExpectError: regexp.MustCompile(`must be at least 1`),
			},
		},
	})
}

// =============================================================================
// OneOf validator tests — volume backup schedule kinds
// =============================================================================

func TestVolumeBackupScheduleResource_invalidKind(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_volume_backup_schedule" "test" {
  volume_instance_id = "vol-inst-123"
  kinds              = ["HOURLY"]
}`,
				ExpectError: regexp.MustCompile(`must be one of`),
			},
		},
	})
}

func TestVolumeBackupScheduleResource_emptyKindsList(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_volume_backup_schedule" "test" {
  volume_instance_id = "vol-inst-123"
  kinds              = []
}`,
				ExpectError: regexp.MustCompile(`must contain at least 1`),
			},
		},
	})
}

// =============================================================================
// List size validator tests — variable_collection must have at least 1 variable
// =============================================================================

func TestVariableCollectionResource_emptyVariablesList(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_variable_collection" "test" {
  environment_id = "00000000-0000-0000-0000-000000000001"
  service_id     = "00000000-0000-0000-0000-000000000002"
  variables      = []
}`,
				ExpectError: regexp.MustCompile(`must contain at least 1`),
			},
		},
	})
}

// =============================================================================
// ConflictsWith validator tests — source_image conflicts with source_repo
// =============================================================================

func TestServiceResource_sourceImageConflictsWithSourceRepo(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name               = "test-svc"
  project_id         = "00000000-0000-0000-0000-000000000001"
  source_image       = "nginx:latest"
  source_repo        = "owner/repo"
  source_repo_branch = "main"
}`,
				ExpectError: regexp.MustCompile(`(?i)cannot be specified when`),
			},
		},
	})
}

func TestServiceInstanceResource_sourceImageConflictsWithSourceRepo(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  source_image   = "nginx:latest"
  source_repo    = "owner/repo"
}`,
				ExpectError: regexp.MustCompile(`(?i)cannot be specified when`),
			},
		},
	})
}

// =============================================================================
// AlsoRequires validator tests — source_repo requires source_repo_branch
// =============================================================================

func TestServiceResource_sourceRepoWithoutBranch(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name        = "test-svc"
  project_id  = "00000000-0000-0000-0000-000000000001"
  source_repo = "owner/repo"
}`,
				ExpectError: regexp.MustCompile(`source_repo_branch`),
			},
		},
	})
}

func TestServiceResource_sourceRepoBranchWithoutRepo(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name               = "test-svc"
  project_id         = "00000000-0000-0000-0000-000000000001"
  source_repo_branch = "main"
}`,
				ExpectError: regexp.MustCompile(`source_repo`),
			},
		},
	})
}

// =============================================================================
// ValidateConfig tests — cross-attribute validation
// =============================================================================

func TestServiceResource_registryUsernameWithoutSourceImage(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name                             = "test-svc"
  project_id                       = "00000000-0000-0000-0000-000000000001"
  source_image_registry_username   = "user"
  source_image_registry_password   = "pass"
}`,
				ExpectError: regexp.MustCompile(`source_image_registry_username.*requires.*source_image`),
			},
		},
	})
}

func TestServiceResource_registryPasswordWithoutSourceImage(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service" "test" {
  name                             = "test-svc"
  project_id                       = "00000000-0000-0000-0000-000000000001"
  source_image_registry_username   = "user"
  source_image_registry_password   = "pass"
}`,
				ExpectError: regexp.MustCompile(`source_image_registry_password.*requires.*source_image`),
			},
		},
	})
}

func TestServiceInstanceResource_buildCommandWithSourceImage(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  source_image   = "nginx:latest"
  build_command  = "npm run build"
}`,
				ExpectError: regexp.MustCompile(`build_command.*cannot be set.*source_image`),
			},
		},
	})
}

func TestServiceInstanceResource_rootDirectoryWithSourceImage(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  source_image   = "nginx:latest"
  root_directory = "/app"
}`,
				ExpectError: regexp.MustCompile(`root_directory.*cannot be set.*source_image`),
			},
		},
	})
}

func TestServiceInstanceResource_configPathWithSourceImage(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  source_image   = "nginx:latest"
  config_path    = "railway.toml"
}`,
				ExpectError: regexp.MustCompile(`config_path.*cannot be set.*source_image`),
			},
		},
	})
}

func TestServiceInstanceResource_cronScheduleWithMultipleReplicas(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  cron_schedule  = "0 */6 * * *"
  num_replicas   = 2
}`,
				ExpectError: regexp.MustCompile(`cron_schedule.*can only be set.*num_replicas.*1`),
			},
		},
	})
}
