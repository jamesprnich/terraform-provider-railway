package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResourceConfigDefault("integration"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_environment.test", "name", "integration"),
					resource.TestCheckResourceAttr("railway_environment.test", "project_id", testAccProjectId),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_environment.test",
				ImportState:       true,
				ImportStateId:     testAccProjectId + ":integration",
				ImportStateVerify: true,
			},
			// Update and Read testing — rename environment
			{
				Config: testAccEnvironmentResourceConfigDefault("integration-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_environment.test", "name", "integration-renamed"),
					resource.TestCheckResourceAttr("railway_environment.test", "project_id", testAccProjectId),
				),
			},
			// Delete testing automatically occurs in TestCase
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
				Config: testAccEnvironmentResourceConfigDefault("disappears-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					testAccCheckEnvironmentDisappears("railway_environment.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccEnvironmentResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_environment" "test" {
  name = "%s"
  project_id = "%s"
}
`, name, testAccProjectId)
}
