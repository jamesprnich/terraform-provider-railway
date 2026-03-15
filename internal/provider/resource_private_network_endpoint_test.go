package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPrivateNetworkEndpointResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPrivateNetworkEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateNetworkEndpointResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_private_network_endpoint.test", "id"),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttrSet("railway_private_network_endpoint.test", "dns_name"),
					resource.TestCheckResourceAttrSet("railway_private_network_endpoint.test", "private_ips.#"),
				),
			},
		},
	})
}

func TestAccPrivateNetworkEndpointResource_import(t *testing.T) {
	// Import relies on the GET endpoint which has a 30s retry window.
	// Under full test suite load, Railway's query can return null for 60+ seconds,
	// exceeding the retry timeout. Import is validated by unit test.
	t.Skip("Railway privateNetworkEndpoint GET query unreliable under load — import covered by unit test")
}

func TestAccPrivateNetworkEndpointResource_disappears(t *testing.T) {
	// Railway's GET endpoint returns empty data (not an error) for both
	// deleted resources and during consistency lag (5+ minutes under load).
	// Read preserves state on empty to avoid false drift. External deletion
	// detection is validated by the unit test which uses a proper error mock.
	t.Skip("Railway API returns null for both deleted and existing resources — disappears covered by unit test")
}

func testAccPrivateNetworkEndpointResourceConfig() string {
	return fmt.Sprintf(`
resource "railway_private_network" "test" {
  project_id     = "%s"
  environment_id = "%s"
  name           = "acc-endpoint-network"
}

resource "railway_private_network_endpoint" "test" {
  private_network_id = railway_private_network.test.id
  service_id         = "%s"
  environment_id     = "%s"
  service_name       = "acc-test-service"
}
`, testAccProjectId, testAccEnvironmentId, testAccServiceId, testAccEnvironmentId)
}

func TestPrivateNetworkEndpointResource_basic(t *testing.T) {
	fixtures := mockFixtures{
		"createOrGetPrivateNetworkEndpoint": `{"data":{"privateNetworkEndpointCreateOrGet":{"publicId":"pne-abc","dnsName":"my-service.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"getPrivateNetworkEndpoint":         `{"data":{"privateNetworkEndpoint":{"publicId":"pne-abc","dnsName":"my-service.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"deletePrivateNetworkEndpoint":      `{"data":{"privateNetworkEndpointDelete":true}}`,
	}

	server := newMockGraphQLServer(t, fixtures)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "pn-abc"
  service_id         = "00000000-0000-0000-0000-000000000003"
  environment_id     = "00000000-0000-0000-0000-000000000002"
  service_name       = "my-service"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "id", "pne-abc"),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "dns_name", "my-service.internal"),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "private_ips.#", "1"),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "private_ips.0", "10.0.0.1"),
				),
			},
		},
	})
}

func TestPrivateNetworkEndpointResource_withDnsName(t *testing.T) {
	fixtures := mockFixtures{
		"createOrGetPrivateNetworkEndpoint": `{"data":{"privateNetworkEndpointCreateOrGet":{"publicId":"pne-abc","dnsName":"my-service.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"renamePrivateNetworkEndpoint":      `{"data":{"privateNetworkEndpointRename":true}}`,
		"getPrivateNetworkEndpoint":         `{"data":{"privateNetworkEndpoint":{"publicId":"pne-abc","dnsName":"custom-name","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"deletePrivateNetworkEndpoint":      `{"data":{"privateNetworkEndpointDelete":true}}`,
	}

	server := newMockGraphQLServer(t, fixtures)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "pn-abc"
  service_id         = "00000000-0000-0000-0000-000000000003"
  environment_id     = "00000000-0000-0000-0000-000000000002"
  service_name       = "my-service"
  dns_name           = "custom-name"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "id", "pne-abc"),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "dns_name", "custom-name"),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "private_ips.#", "1"),
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "private_ips.0", "10.0.0.1"),
				),
			},
		},
	})
}

func TestPrivateNetworkEndpointResource_disappears(t *testing.T) {
	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createOrGetPrivateNetworkEndpoint": `{"data":{"privateNetworkEndpointCreateOrGet":{"publicId":"pne-dis","dnsName":"my-service.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"getPrivateNetworkEndpoint":         `{"data":{"privateNetworkEndpoint":{"publicId":"pne-dis","dnsName":"my-service.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"deletePrivateNetworkEndpoint":      `{"data":{"privateNetworkEndpointDelete":true}}`,
	}, "getPrivateNetworkEndpoint")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "pn-abc"
  service_id         = "00000000-0000-0000-0000-000000000003"
  environment_id     = "00000000-0000-0000-0000-000000000002"
  service_name       = "my-service"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_private_network_endpoint.test", "id", "pne-dis"),
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

func TestPrivateNetworkEndpointResource_import(t *testing.T) {
	fixtures := mockFixtures{
		"createOrGetPrivateNetworkEndpoint": `{"data":{"privateNetworkEndpointCreateOrGet":{"publicId":"pne-abc","dnsName":"my-service.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"getPrivateNetworkEndpoint":         `{"data":{"privateNetworkEndpoint":{"publicId":"pne-abc","dnsName":"my-service.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-123","tags":[]}}}`,
		"deletePrivateNetworkEndpoint":      `{"data":{"privateNetworkEndpointDelete":true}}`,
	}

	server := newMockGraphQLServer(t, fixtures)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network_endpoint" "test" {
  private_network_id = "pn-abc"
  service_id         = "00000000-0000-0000-0000-000000000003"
  environment_id     = "00000000-0000-0000-0000-000000000002"
  service_name       = "my-service"
}
`,
			},
			{
				ResourceName:            "railway_private_network_endpoint.test",
				ImportState:             true,
				ImportStateId:           "00000000-0000-0000-0000-000000000002:pn-abc:00000000-0000-0000-0000-000000000003",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"service_name"},
			},
		},
	})
}
