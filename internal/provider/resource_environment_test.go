package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// railway_environment acceptance tests (v0.11.0 shape)
//
// Under strict_env_scoping = true (provider default), source_environment_id
// is required — omitting it fails at plan time. Setting it creates the env
// as a fork of the referenced env.
// =============================================================================

func TestAccEnvironmentResource_asFork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_fork("integration"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_environment.test", "name", "integration"),
					resource.TestCheckResourceAttr("railway_environment.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_environment.test", "source_environment_id", testAccEnvironmentId),
				),
			},
			{
				ResourceName:      "railway_environment.test",
				ImportState:       true,
				ImportStateId:     testAccProjectId + ":integration",
				ImportStateVerify: true,
			},
			{
				Config: testAccEnvironmentResourceConfig_fork("integration-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_environment.test", "name", "integration-renamed"),
				),
			},
		},
	})
}

// TestAccEnvironmentResource_strictModeRequiresFork verifies the strict-mode
// diagnostic fires when source_environment_id is missing. Runs on a plan (no
// mutations executed) so no cooldown applies.
func TestAccEnvironmentResource_strictModeRequiresFork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "railway_environment" "test" {
  name       = "strict-diag"
  project_id = "%s"
}
`, testAccProjectId),
				ExpectError: regexp.MustCompile(`source_environment_id is required`),
			},
		},
	})
}

// TestAccEnvironmentResource_permissiveNonFork verifies opting out of strict
// mode lets you create a non-fork environment (pre-v0.11.0 behaviour). Also
// verifies source_environment_id is null in state when unset.
func TestAccEnvironmentResource_permissiveNonFork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "railway" {
  strict_env_scoping = false
}

resource "railway_environment" "test" {
  name       = "permissive-nonfork"
  project_id = "%s"
}
`, testAccProjectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_environment.test", "name", "permissive-nonfork"),
					resource.TestCheckNoResourceAttr("railway_environment.test", "source_environment_id"),
				),
			},
		},
	})
}

func TestAccEnvironmentResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_fork("disappears-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					testAccCheckEnvironmentDisappears("railway_environment.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccEnvironmentResourceConfig_fork(name string) string {
	return fmt.Sprintf(`
resource "railway_environment" "test" {
  name                  = "%s"
  project_id            = "%s"
  source_environment_id = "%s"
}
`, name, testAccProjectId, testAccEnvironmentId)
}
