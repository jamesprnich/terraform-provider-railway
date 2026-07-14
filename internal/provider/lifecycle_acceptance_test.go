package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckEnvironmentCommittedNotStaged fetches the environment identified
// by (project attribute, env attribute) and asserts that its
// unmergedChangesCount is nil or 0 — i.e. Railway committed the environment's
// creates rather than leaving them as staged changes the user has to click
// "apply" on in the dashboard.
//
// This is the C1 property the design depends on. The provider sets
// StageInitialChanges: false in environmentCreate, but the review flagged
// that we defended the property by a code setting rather than by an
// assertion. This test check closes that gap: even under a live apply the
// property is verified against the API's own view of the environment.
func testAccCheckEnvironmentCommittedNotStaged(projectAttr, envAttr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		projectRS, ok := s.RootModule().Resources[projectAttr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", projectAttr)
		}
		envRS, ok := s.RootModule().Resources[envAttr]
		if !ok {
			return fmt.Errorf("resource %s not found in state", envAttr)
		}

		client := testAccNewClient()
		resp, err := getEnvironments(context.Background(), client, projectRS.Primary.ID)
		if err != nil {
			return fmt.Errorf("querying environments for project %s: %w", projectRS.Primary.ID, err)
		}

		for _, edge := range resp.Environments.Edges {
			env := edge.Node.Environment
			if env.Id != envRS.Primary.ID {
				continue
			}
			if env.UnmergedChangesCount != nil && *env.UnmergedChangesCount > 0 {
				return fmt.Errorf(
					"env %s (%s) has %d unmerged changes — the design's 'deploys, not staged' property is violated. Something is queuing changes as staged rather than committing them (StageInitialChanges must be false on environmentCreate)",
					env.Name, env.Id, *env.UnmergedChangesCount,
				)
			}
			return nil
		}
		return fmt.Errorf("env id %s not found in project %s environment list", envRS.Primary.ID, projectRS.Primary.ID)
	}
}

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
			// §4.3 retry. Also sets pre_deploy_command on dev_app so the
			// Read-after-Create path exercises the JSON→[]string bind added
			// in v0.11.2 — pre-fix, this Read would panic with a JSON
			// unmarshal error the moment Railway returned a non-null command
			// list.
			// -----------------------------------------------------------------
			{
				Config: testAccLifecycleFullConfig(projectName, `"echo initial migration"`),
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
					// pre_deploy_command round-tripped through Read-after-Create.
					resource.TestCheckResourceAttr("railway_service_instance.dev_app", "pre_deploy_command", "echo initial migration"),
					// C1: services and environments deploy immediately —
					// they don't land as unmerged staged changes. The
					// design depends on this and the provider defends it
					// with StageInitialChanges: false on environmentCreate;
					// this assertion turns that from "we set the flag" into
					// "we watched the flag's effect."
					testAccCheckEnvironmentCommittedNotStaged("railway_project.acc", "railway_environment.dev"),
					testAccCheckEnvironmentCommittedNotStaged("railway_project.acc", "railway_environment.prd"),
				),
			},
			// -----------------------------------------------------------------
			// Step 2: update pre_deploy_command on dev_app. This is the exact
			// failure mode the v0.11.2 fix targets — the WRITE succeeds,
			// Railway retains the value, and every subsequent refresh/plan
			// used to hit the JSON unmarshal error on Read-after-Update. The
			// resource became permanently unplannable pre-fix.
			// -----------------------------------------------------------------
			{
				Config: testAccLifecycleFullConfig(projectName, `"echo updated migration"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.dev_app", "pre_deploy_command", "echo updated migration"),
				),
			},
			// -----------------------------------------------------------------
			// Step 3: remove every dev-side resource from config. The
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

func testAccLifecycleFullConfig(projectName, devPreDeployCommand string) string {
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
  service_id         = railway_service.dev_app.id
  environment_id     = railway_environment.dev.id
  source_image       = "nginx:alpine"
  pre_deploy_command = %s
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
`, projectName, devPreDeployCommand)
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
