package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomDomainResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomDomainDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing. The CNAME + verification fields all come
			// from Railway's status block on the customDomain response. Before
			// v0.11.4 the provider took dnsRecords[0] and discarded verification
			// info entirely; this test asserts the full replacement contract
			// against a real Railway workspace.
			{
				Config: testAccCustomDomainResourceConfigDefault("terraform.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_custom_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "domain", "terraform.example.com"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "terraform"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "zone", "example.com"),
					resource.TestCheckResourceAttrSet("railway_custom_domain.test", "dns_record_value"),
					// A fresh custom domain hasn't had its TXT set up, so verified
					// must be false. The verification host + token must both be
					// present so the consumer can build the TXT record.
					resource.TestCheckResourceAttr("railway_custom_domain.test", "verified", "false"),
					resource.TestCheckResourceAttrSet("railway_custom_domain.test", "verification_dns_host"),
					resource.TestCheckResourceAttrSet("railway_custom_domain.test", "verification_token"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_custom_domain.test",
				ImportState:       true,
				ImportStateId:     testAccServiceId + ":" + testAccEnvironmentName + ":terraform.example.com",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccCustomDomainResourceConfigDefault("terraform.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_custom_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "domain", "terraform.example.com"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "terraform"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "zone", "example.com"),
					resource.TestCheckResourceAttrSet("railway_custom_domain.test", "dns_record_value"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccCustomDomainResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomDomainResourceConfigDefault("terraform-disappears.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_custom_domain.test", "id", uuidRegex()),
					testAccCheckCustomDomainDisappears("railway_custom_domain.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCustomDomainResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_custom_domain" "test" {
  domain = "%s"
  environment_id = "%s"
  service_id = "%s"
}
`, name, testAccEnvironmentId, testAccServiceId)
}
