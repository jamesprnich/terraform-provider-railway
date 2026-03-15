package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableCollectionResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVariableCollectionDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVariableCollectionResourceConfigDefault("one", "two", "three"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", testAccServiceId+":"+testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.name", "VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.value", "one"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.name", "VALUE_B"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.value", "two"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.name", "VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.value", "three"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable_collection.test",
				ImportState:       true,
				ImportStateId:     testAccServiceId + ":" + testAccEnvironmentName + ":VALUE_A:VALUE_B:VALUE_C",
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccVariableCollectionResourceConfigDefault("four", "five", "six"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", testAccServiceId+":"+testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.name", "VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.value", "four"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.name", "VALUE_B"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.value", "five"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.name", "VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.value", "six"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable_collection.test",
				ImportState:       true,
				ImportStateId:     testAccServiceId + ":" + testAccEnvironmentName + ":VALUE_A:VALUE_B:VALUE_C",
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccVariableCollectionResourceConfigNonDefault("four", "five", "six"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", testAccServiceId+":"+testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.name", "VALUE_B"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.value", "four"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.name", "VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.value", "five"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.name", "VALUE_D"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.value", "six"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable_collection.test",
				ImportState:       true,
				ImportStateId:     testAccServiceId + ":" + testAccEnvironmentName + ":VALUE_B:VALUE_C:VALUE_D",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccVariableCollectionResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVariableCollectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVariableCollectionResourceConfigDefault("a", "b", "c"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.#", "3"),
					testAccCheckVariableCollectionDisappears("railway_variable_collection.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccVariableCollectionResourceConfigDefault(valueA, valueB, valueC string) string {
	return fmt.Sprintf(`
resource "railway_variable_collection" "test" {
  environment_id = "%s"
  service_id = "%s"

  variables = [
    {
      name = "VALUE_A"
      value = "%s"
    },
    {
      name = "VALUE_B"
      value = "%s"
    },
    {
      name = "VALUE_C"
      value = "%s"
    }
  ]
}
`, testAccEnvironmentId, testAccServiceId, valueA, valueB, valueC)
}

func testAccVariableCollectionResourceConfigNonDefault(valueB, valueC, valueD string) string {
	return fmt.Sprintf(`
resource "railway_variable_collection" "test" {
  environment_id = "%s"
  service_id = "%s"

  variables = [
    {
      name = "VALUE_B"
      value = "%s"
    },
    {
      name = "VALUE_C"
      value = "%s"
    },
    {
      name = "VALUE_D"
      value = "%s"
    }
  ]
}
`, testAccEnvironmentId, testAccServiceId, valueB, valueC, valueD)
}
