package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// railway_service acceptance tests (v0.11.0 shape)
//
// Runs against a live Railway API when TF_ACC=1 and RAILWAY_TOKEN are set.
// The shell shape is name + project_id + environment_id + optional volume +
// optional icon — every configuration field (source, cron, build, replicas,
// registry creds, etc.) belongs on railway_service_instance now.
//
// Full pre-v0.11.0 test coverage was removed with the fields; the CHANGELOG
// documents the migration.
// =============================================================================

func TestAccServiceResource_basic(t *testing.T) {
	name := "tf-acc-svc-" + acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service.test", "name", name),
					resource.TestCheckResourceAttrSet("railway_service.test", "id"),
					resource.TestCheckResourceAttrSet("railway_service.test", "project_id"),
					resource.TestCheckResourceAttrSet("railway_service.test", "environment_id"),
					// Shell shape — no source_* / cron / regions / registry
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccServiceResource_renameAndIcon(t *testing.T) {
	name := "tf-acc-svc-" + acctest.RandString(6)
	renamed := name + "-r"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceResourceConfig_basic(name),
				Check:  resource.TestCheckResourceAttr("railway_service.test", "name", name),
			},
			{
				Config: testAccServiceResourceConfig_withIcon(renamed, "🐹"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service.test", "name", renamed),
					resource.TestCheckResourceAttr("railway_service.test", "icon", "🐹"),
				),
			},
		},
	})
}

func TestAccServiceResource_withInlineVolume(t *testing.T) {
	name := "tf-acc-svc-" + acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceResourceConfig_withVolume(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service.test", "name", name),
					resource.TestCheckResourceAttrSet("railway_service.test", "volume.id"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "data"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/data"),
				),
			},
		},
	})
}

func TestAccServiceResource_disappears(t *testing.T) {
	name := "tf-acc-svc-" + acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceResourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServiceDisappears("railway_service.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// -----------------------------------------------------------------------------
// HCL config builders — each creates a fork env off the fixture env, then
// a service scoped to that fork. This is the v0.11.0 strict-mode pattern.
// -----------------------------------------------------------------------------

func testAccServiceResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "railway_environment" "test" {
  name                  = "%s-env"
  project_id            = "%s"
  source_environment_id = "%s"
}

resource "railway_service" "test" {
  name           = "%s"
  project_id     = "%s"
  environment_id = railway_environment.test.id
}
`, name, testAccProjectId, testAccEnvironmentId, name, testAccProjectId)
}

func testAccServiceResourceConfig_withIcon(name, icon string) string {
	return fmt.Sprintf(`
resource "railway_environment" "test" {
  name                  = "%s-env"
  project_id            = "%s"
  source_environment_id = "%s"
}

resource "railway_service" "test" {
  name           = "%s"
  project_id     = "%s"
  environment_id = railway_environment.test.id
  icon           = "%s"
}
`, name, testAccProjectId, testAccEnvironmentId, name, testAccProjectId, icon)
}

func testAccServiceResourceConfig_withVolume(name string) string {
	return fmt.Sprintf(`
resource "railway_environment" "test" {
  name                  = "%s-env"
  project_id            = "%s"
  source_environment_id = "%s"
}

resource "railway_service" "test" {
  name           = "%s"
  project_id     = "%s"
  environment_id = railway_environment.test.id

  volume = {
    name       = "data"
    mount_path = "/data"
  }
}
`, name, testAccProjectId, testAccEnvironmentId, name, testAccProjectId)
}
