package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSharedVariableResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSharedVariableDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSharedVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", testAccProjectId+":"+testAccEnvironmentId+":API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", testAccProjectId),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_shared_variable.test",
				ImportState:       true,
				ImportStateId:     testAccProjectId + ":" + testAccEnvironmentName + ":API_KEY",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccSharedVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", testAccProjectId+":"+testAccEnvironmentId+":API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", testAccProjectId),
				),
			},
			// Update and Read testing
			{
				Config: testAccSharedVariableResourceConfigDefault("nice"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", testAccProjectId+":"+testAccEnvironmentId+":API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "nice"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", testAccProjectId),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_shared_variable.test",
				ImportState:       true,
				ImportStateId:     testAccProjectId + ":" + testAccEnvironmentName + ":API_KEY",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSharedVariableResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSharedVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSharedVariableResourceConfigDefault("disappears-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					testAccCheckSharedVariableDisappears("railway_shared_variable.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccSharedVariableResourceConfigDefault(value string) string {
	return fmt.Sprintf(`
resource "railway_shared_variable" "test" {
  name = "API_KEY"
  value = "%s"
  environment_id = "%s"
  project_id = "%s"
}
`, value, testAccEnvironmentId, testAccProjectId)
}
