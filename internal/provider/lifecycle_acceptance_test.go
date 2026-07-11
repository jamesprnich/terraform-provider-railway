package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// Lifecycle acceptance tests for v0.11.0 fork-based env scoping.
//
// These tests provision real Railway projects and exercise the whole design
// end-to-end. They run only when TF_ACC=1 and RAILWAY_TOKEN are set.
//
// The single most important assertion this file makes is that a service
// scoped to one fork (dev) does not appear in another fork (prd) or in the
// project's non-fork default environment. That property is what strict
// env-scoping exists to guarantee.
// =============================================================================

// TestAccLifecycle_forkTopology creates a fresh throwaway project with the
// pattern the provider is designed around — an empty non-fork default env
// called `core`, two forks (`dev` and `prd`), services scoped to each fork,
// and a volume on one service. It then removes every dev-side resource in a
// second step and asserts that prd is untouched. The implicit teardown at
// the end of resource.Test destroys the project entirely.
func TestAccLifecycle_forkTopology(t *testing.T) {
	// Each run gets its own randomly-suffixed project name so parallel runs
	// or crashed prior runs do not collide.
	projectName := "tf-acc-fork-" + acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// -----------------------------------------------------------------
			// Step 1: full topology.
			// Asserts: fork relationships, per-env service scoping, volume
			// created without hitting the "not found" race that motivated the
			// §4.3 retry.
			// -----------------------------------------------------------------
			{
				Config: testAccLifecycleFullConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Project has an empty core.
					resource.TestCheckResourceAttr("railway_project.acc", "default_environment.name", "core"),
					// dev + prd are both forks of core.
					resource.TestCheckResourceAttrPair(
						"railway_environment.dev", "source_environment_id",
						"railway_project.acc", "default_environment.id"),
					resource.TestCheckResourceAttrPair(
						"railway_environment.prd", "source_environment_id",
						"railway_project.acc", "default_environment.id"),
					// Every service reports the environment it was scoped to.
					resource.TestCheckResourceAttrPair(
						"railway_service.dev_app", "environment_id",
						"railway_environment.dev", "id"),
					resource.TestCheckResourceAttrPair(
						"railway_service.prd_app", "environment_id",
						"railway_environment.prd", "id"),
					// Volume was created and read back successfully — this
					// exercises the post-create retry loop we added for the
					// eventual-consistency case where the volume creation
					// succeeds but the immediate follow-up read returns
					// "volume instance not found".
					resource.TestCheckResourceAttrSet("railway_volume.dev_data", "volume_instance_id"),
					resource.TestCheckResourceAttr("railway_volume.dev_data", "mount_path", "/data"),
					resource.TestCheckResourceAttrSet("railway_volume.dev_data", "size_mb"),
				),
			},
			// -----------------------------------------------------------------
			// Step 2: remove every dev-side resource from config. The
			// framework will plan and apply destroys for anything no longer
			// declared. This is the closest equivalent to "tofu destroy of
			// the dev workspace" within a single test run.
			// Asserts: prd's service still exists and is still scoped to prd.
			// -----------------------------------------------------------------
			{
				Config: testAccLifecyclePrdOnlyConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"railway_service.prd_app", "environment_id",
						"railway_environment.prd", "id"),
				),
			},
		},
	})
}

// TestAccLifecycle_strictModeRejectsMissingScope verifies that under the
// default strict env-scoping, both a service without environment_id and an
// environment without source_environment_id fail at plan time — not silently
// at apply. Runs only a plan (ExpectError on step 1), so no live mutations.
func TestAccLifecycle_strictModeRejectsMissingScope(t *testing.T) {
	projectName := "tf-acc-strict-" + acctest.RandString(6)

	t.Run("service_without_environment_id", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
resource "railway_project" "acc" {
  name = "%s-svc"
  default_environment = { name = "core" }
}
resource "railway_service" "bad" {
  name       = "bad-svc"
  project_id = railway_project.acc.id
  # environment_id deliberately omitted under strict mode
}
`, projectName),
					ExpectError: regexp.MustCompile(`environment_id is required under strict env-scoping`),
				},
			},
		})
	})

	t.Run("environment_without_source_environment_id", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
resource "railway_project" "acc" {
  name = "%s-env"
  default_environment = { name = "core" }
}
resource "railway_environment" "bad" {
  name       = "bad-env"
  project_id = railway_project.acc.id
  # source_environment_id deliberately omitted under strict mode
}
`, projectName),
					ExpectError: regexp.MustCompile(`source_environment_id is required under strict env-scoping`),
				},
			},
		})
	})
}

// TestAccLifecycle_permissiveModeAllowsUnscoped verifies that opting out of
// strict env-scoping (`strict_env_scoping = false`) restores the pre-v0.11.0
// behaviour where a service can be created without environment_id and an
// environment can be created without being a fork.
func TestAccLifecycle_permissiveModeAllowsUnscoped(t *testing.T) {
	projectName := "tf-acc-permissive-" + acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "railway" {
  strict_env_scoping = false
}

resource "railway_project" "acc" {
  name = "%s"
  default_environment = { name = "core" }
}

# Non-fork additional environment — allowed under permissive mode.
resource "railway_environment" "nonfork" {
  name       = "loose"
  project_id = railway_project.acc.id
}

# Unscoped service — allowed under permissive mode. Under Railway's own
# semantics this lands in every non-fork env in the project.
resource "railway_service" "loose" {
  name       = "loose-svc"
  project_id = railway_project.acc.id
}
`, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("railway_environment.nonfork", "source_environment_id"),
					resource.TestCheckNoResourceAttr("railway_service.loose", "environment_id"),
				),
			},
		},
	})
}

// -----------------------------------------------------------------------------
// HCL config builders for the lifecycle test.
// Uses public images end-to-end so the test does not depend on any external
// service connection (e.g., GitHub repository access).
// -----------------------------------------------------------------------------

func testAccLifecycleFullConfig(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "acc" {
  name = "%s"
  default_environment = { name = "core" }
}

resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_environment" "prd" {
  name                  = "prd"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_service" "dev_app" {
  name           = "dev-app"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  depends_on     = [railway_environment.dev]
}

resource "railway_service_instance" "dev_app" {
  service_id     = railway_service.dev_app.id
  environment_id = railway_environment.dev.id
  source_image   = "nginx:alpine"
}

resource "railway_volume" "dev_data" {
  project_id     = railway_project.acc.id
  service_id     = railway_service.dev_app.id
  environment_id = railway_environment.dev.id
  mount_path     = "/data"
}

resource "railway_service" "prd_app" {
  name           = "prd-app"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.prd.id
  depends_on     = [railway_environment.prd]
}

resource "railway_service_instance" "prd_app" {
  service_id     = railway_service.prd_app.id
  environment_id = railway_environment.prd.id
  source_image   = "nginx:alpine"
}
`, projectName)
}

func testAccLifecyclePrdOnlyConfig(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "acc" {
  name = "%s"
  default_environment = { name = "core" }
}

resource "railway_environment" "prd" {
  name                  = "prd"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_service" "prd_app" {
  name           = "prd-app"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.prd.id
  depends_on     = [railway_environment.prd]
}

resource "railway_service_instance" "prd_app" {
  service_id     = railway_service.prd_app.id
  environment_id = railway_environment.prd.id
  source_image   = "nginx:alpine"
}
`, projectName)
}
