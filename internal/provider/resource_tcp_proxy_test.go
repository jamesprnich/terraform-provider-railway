package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccTcpProxyResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTcpProxyResourceConfigDefault(6379),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_tcp_proxy.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "application_port", "6379"),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "proxy_port"),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "domain"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_tcp_proxy.test",
				ImportState:       true,
				ImportStateIdFunc: tcpProxyImportIdFunc,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccTcpProxyResourceConfigDefault(6379),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_tcp_proxy.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "application_port", "6379"),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "proxy_port"),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "domain"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTcpProxyResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTcpProxyResourceConfigDefault(5432),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_tcp_proxy.test", "id", uuidRegex()),
					testAccCheckTcpProxyDisappears("railway_tcp_proxy.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccTcpProxyResourceConfigDefault(port int) string {
	return fmt.Sprintf(`
resource "railway_tcp_proxy" "test" {
  application_port = "%d"
  environment_id = "%s"
  service_id = "%s"
}
`, port, testAccEnvironmentId, testAccServiceId)
}

func tcpProxyImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_tcp_proxy.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s:%s", rawState.Primary.Attributes["service_id"], rawState.Primary.Attributes["environment_id"], rawState.Primary.Attributes["id"]), nil
}
