package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

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
