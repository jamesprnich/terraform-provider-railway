package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", testAccServiceId+":"+testAccEnvironmentId+":REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", testAccProjectId),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable.test",
				ImportState:       true,
				ImportStateId:     testAccServiceId + ":" + testAccEnvironmentName + ":REDIS_URL",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", testAccServiceId+":"+testAccEnvironmentId+":REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", testAccProjectId),
				),
			},
			// Update and Read testing
			{
				Config: testAccVariableResourceConfigDefault("$${{redis.REDIS_URL}}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", testAccServiceId+":"+testAccEnvironmentId+":REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "${{redis.REDIS_URL}}"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", testAccProjectId),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable.test",
				ImportState:       true,
				ImportStateId:     testAccServiceId + ":" + testAccEnvironmentName + ":REDIS_URL",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccVariableResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVariableResourceConfigDefault("disappears-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "name", "REDIS_URL"),
					testAccCheckVariableDisappears("railway_variable.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccVariableResourceConfigDefault(value string) string {
	return fmt.Sprintf(`
resource "railway_variable" "test" {
  name = "REDIS_URL"
  value = "%s"
  environment_id = "%s"
  service_id = "%s"
}
`, value, testAccEnvironmentId, testAccServiceId)
}
