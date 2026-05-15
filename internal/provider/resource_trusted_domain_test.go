package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance test is intentionally skipped — trusted domains are WORKSPACE-LEVEL
// resources. Creating one against the live API would modify the SSO configuration
// of the test workspace, affecting every project (not just the fixture project)
// and persisting after fixture cleanup. Requires explicit user approval and manual
// validation in an isolated workspace to enable.
func TestAccTrustedDomainResourceDefault(t *testing.T) {
	t.Skip("workspace-level resource — would persist outside the fixture project. Test manually in an isolated workspace.")
}

func TestTrustedDomainResource_basic(t *testing.T) {
	workspaceId := "ws-abc-123"

	server := newMockGraphQLServer(t, mockFixtures{
		"createTrustedDomain": `{"data":{"trustedDomainCreate":{"id":"td-1","domainName":"example.com","role":"MEMBER","status":"PENDING","workspaceId":"` + workspaceId + `"}}}`,
		"getTrustedDomains":   `{"data":{"trustedDomains":{"edges":[{"node":{"id":"td-1","domainName":"example.com","role":"MEMBER","status":"PENDING","workspaceId":"` + workspaceId + `"}}]}}}`,
		"deleteTrustedDomain": `{"data":{"trustedDomainDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_trusted_domain" "test" {
  workspace_id = "` + workspaceId + `"
  domain_name  = "example.com"
  role         = "MEMBER"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_trusted_domain.test", "id", "td-1"),
					resource.TestCheckResourceAttr("railway_trusted_domain.test", "workspace_id", workspaceId),
					resource.TestCheckResourceAttr("railway_trusted_domain.test", "domain_name", "example.com"),
					resource.TestCheckResourceAttr("railway_trusted_domain.test", "role", "MEMBER"),
					resource.TestCheckResourceAttr("railway_trusted_domain.test", "status", "PENDING"),
				),
			},
		},
	})
}

func TestTrustedDomainResource_disappears(t *testing.T) {
	workspaceId := "ws-abc-123"

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createTrustedDomain": `{"data":{"trustedDomainCreate":{"id":"td-2","domainName":"example.com","role":"MEMBER","status":"VERIFIED","workspaceId":"` + workspaceId + `"}}}`,
		"getTrustedDomains":   `{"data":{"trustedDomains":{"edges":[{"node":{"id":"td-2","domainName":"example.com","role":"MEMBER","status":"VERIFIED","workspaceId":"` + workspaceId + `"}}]}}}`,
		"deleteTrustedDomain": `{"data":{"trustedDomainDelete":true}}`,
	}, "getTrustedDomains")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_trusted_domain" "test" {
  workspace_id = "` + workspaceId + `"
  domain_name  = "example.com"
  role         = "MEMBER"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_trusted_domain.test", "id", "td-2"),
					func(s *terraform.State) error {
						disappear()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
