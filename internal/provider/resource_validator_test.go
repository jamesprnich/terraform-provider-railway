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

// Note (v0.11.0): cron_schedule, source_*, registry_*, and regions were
// removed from railway_service — they belong on railway_service_instance,
// which is the env-scoped shape Railway's API canonically models. The
// TestServiceResource_cronScheduleTooShort / _sourceRepoWithoutBranch /
// _sourceImageConflictsWithSourceRepo / _registryUsernameWithoutSourceImage
// / _registryPasswordWithoutSourceImage / _replicasTooLow tests were
// removed with the fields. Equivalent validation now lives on
// railway_service_instance (see TestServiceInstanceResource_* below).

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

// TestServiceResource_replicasTooLow removed in v0.11.0 along with the
// `regions` field. num_replicas validation now lives on
// railway_service_instance.

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

// TestServiceResource_sourceImageConflictsWithSourceRepo removed in v0.11.0
// — source_* fields are only on railway_service_instance now.

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

// TestServiceResource_sourceRepoWithoutBranch and
// TestServiceResource_sourceRepoBranchWithoutRepo removed in v0.11.0 —
// source_repo* fields are only on railway_service_instance now.

// =============================================================================
// ValidateConfig tests — cross-attribute validation
// =============================================================================

// TestServiceResource_registryUsernameWithoutSourceImage and
// TestServiceResource_registryPasswordWithoutSourceImage removed in v0.11.0 —
// registry_credentials is only on railway_service_instance now.

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
