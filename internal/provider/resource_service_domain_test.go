package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccServiceDomainResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDomainDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceDomainResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttrSet("railway_service_domain.test", "subdomain"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_service_domain.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_service_domain.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttrSet("railway_service_domain.test", "domain"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "suffix", "up.railway.app"),
				),
			},
			// ImportState testing
			{
				ResourceName:  "railway_service_domain.test",
				ImportState:   true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["railway_service_domain.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					domain := rs.Primary.Attributes["domain"]
					return testAccServiceId + ":" + testAccEnvironmentName + ":" + domain, nil
				},
				ImportStateVerify: true,
			},
			// Idempotency check
			{
				Config: testAccServiceDomainResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttrSet("railway_service_domain.test", "subdomain"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "suffix", "up.railway.app"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceDomainResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceDomainResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					testAccCheckServiceDomainDisappears("railway_service_domain.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccServiceDomainResourceConfig() string {
	return fmt.Sprintf(`
resource "railway_service_domain" "test" {
  environment_id = "%s"
  service_id = "%s"
}
`, testAccEnvironmentId, testAccServiceId)
}
