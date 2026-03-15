package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPrivateNetworkResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPrivateNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateNetworkResourceConfig("acc-test-network"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_private_network.test", "id"),
					resource.TestCheckResourceAttr("railway_private_network.test", "name", "acc-test-network"),
					resource.TestCheckResourceAttr("railway_private_network.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_private_network.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttrSet("railway_private_network.test", "dns_name"),
					resource.TestCheckResourceAttrSet("railway_private_network.test", "network_id"),
				),
			},
			// Import
			{
				ResourceName: "railway_private_network.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["railway_private_network.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return testAccEnvironmentId + ":" + rs.Primary.ID, nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPrivateNetworkResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPrivateNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateNetworkResourceConfig("acc-disappears-network"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_private_network.test", "id"),
					testAccCheckPrivateNetworkDisappears("railway_private_network.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccPrivateNetworkResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "railway_private_network" "test" {
  project_id     = "%s"
  environment_id = "%s"
  name           = "%s"
}
`, testAccProjectId, testAccEnvironmentId, name)
}

func TestPrivateNetworkResource_basic(t *testing.T) {
	fixtures := mockFixtures{
		"createOrGetPrivateNetwork": `{"data":{"privateNetworkCreateOrGet":{"publicId":"pn-abc","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":[]}}}`,
		"getPrivateNetworks":        `{"data":{"privateNetworks":[{"publicId":"pn-abc","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":[]}]}}`,
		"deletePrivateNetworksForEnvironment": `{"data":{"privateNetworksForEnvironmentDelete":true}}`,
	}

	server := newMockGraphQLServer(t, fixtures)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network" "test" {
  project_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  name           = "test-network"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_private_network.test", "id", "pn-abc"),
					resource.TestCheckResourceAttr("railway_private_network.test", "dns_name", "test-network.internal"),
					resource.TestCheckResourceAttr("railway_private_network.test", "network_id", "42"),
					resource.TestCheckResourceAttr("railway_private_network.test", "name", "test-network"),
				),
			},
		},
	})
}

func TestPrivateNetworkResource_withTags(t *testing.T) {
	fixtures := mockFixtures{
		"createOrGetPrivateNetwork": `{"data":{"privateNetworkCreateOrGet":{"publicId":"pn-abc","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":["tag1","tag2"]}}}`,
		"getPrivateNetworks":        `{"data":{"privateNetworks":[{"publicId":"pn-abc","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":["tag1","tag2"]}]}}`,
		"deletePrivateNetworksForEnvironment": `{"data":{"privateNetworksForEnvironmentDelete":true}}`,
	}

	server := newMockGraphQLServer(t, fixtures)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network" "test" {
  project_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  name           = "test-network"
  tags           = ["tag1", "tag2"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_private_network.test", "id", "pn-abc"),
					resource.TestCheckResourceAttr("railway_private_network.test", "dns_name", "test-network.internal"),
					resource.TestCheckResourceAttr("railway_private_network.test", "network_id", "42"),
					resource.TestCheckResourceAttr("railway_private_network.test", "name", "test-network"),
					resource.TestCheckResourceAttr("railway_private_network.test", "tags.#", "2"),
					resource.TestCheckResourceAttr("railway_private_network.test", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("railway_private_network.test", "tags.1", "tag2"),
				),
			},
		},
	})
}

func TestPrivateNetworkResource_disappears(t *testing.T) {
	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createOrGetPrivateNetwork":           `{"data":{"privateNetworkCreateOrGet":{"publicId":"pn-dis","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":[]}}}`,
		"getPrivateNetworks":                  `{"data":{"privateNetworks":[{"publicId":"pn-dis","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":[]}]}}`,
		"deletePrivateNetworksForEnvironment": `{"data":{"privateNetworksForEnvironmentDelete":true}}`,
	}, "getPrivateNetworks")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_private_network" "test" {
  project_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  name           = "test-network"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_private_network.test", "id", "pn-dis"),
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

func TestPrivateNetworkResource_import(t *testing.T) {
	fixtures := mockFixtures{
		"createOrGetPrivateNetwork": `{"data":{"privateNetworkCreateOrGet":{"publicId":"pn-abc","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":[]}}}`,
		"getPrivateNetworks":        `{"data":{"privateNetworks":[{"publicId":"pn-abc","projectId":"00000000-0000-0000-0000-000000000001","environmentId":"00000000-0000-0000-0000-000000000002","name":"test-network","dnsName":"test-network.internal","networkId":42,"tags":[]}]}}`,
		"deletePrivateNetworksForEnvironment": `{"data":{"privateNetworksForEnvironmentDelete":true}}`,
	}

	server := newMockGraphQLServer(t, fixtures)
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_private_network" "test" {
  project_id     = "00000000-0000-0000-0000-000000000001"
  environment_id = "00000000-0000-0000-0000-000000000002"
  name           = "test-network"
}
`,
			},
			{
				ResourceName:      "railway_private_network.test",
				ImportState:       true,
				ImportStateId:     "00000000-0000-0000-0000-000000000002:pn-abc",
				ImportStateVerify: true,
			},
		},
	})
}
