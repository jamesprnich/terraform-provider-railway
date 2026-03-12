package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEgressGatewayResource_basic(t *testing.T) {
	server := newMockGraphQLServer(t, mockFixtures{
		"createEgressGateway": `{"data":{"egressGatewayAssociationCreate":[{"ipv4":"1.2.3.4","region":"us-west1"}]}}`,
		"getEgressGateways":   `{"data":{"egressGateways":[{"ipv4":"1.2.3.4","region":"us-west1"}]}}`,
		"clearEgressGateways": `{"data":{"egressGatewayAssociationsClear":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "b5b3e3a0-1234-5678-9abc-def012345678"
  environment_id = "c6c4f4b1-2345-6789-abcd-ef0123456789"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "id", "b5b3e3a0-1234-5678-9abc-def012345678:c6c4f4b1-2345-6789-abcd-ef0123456789"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "service_id", "b5b3e3a0-1234-5678-9abc-def012345678"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "environment_id", "c6c4f4b1-2345-6789-abcd-ef0123456789"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "ip_addresses.#", "1"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "ip_addresses.0", "1.2.3.4"),
				),
			},
		},
	})
}

func TestEgressGatewayResource_withRegion(t *testing.T) {
	server := newMockGraphQLServer(t, mockFixtures{
		"createEgressGateway": `{"data":{"egressGatewayAssociationCreate":[{"ipv4":"5.6.7.8","region":"eu-west1"}]}}`,
		"getEgressGateways":   `{"data":{"egressGateways":[{"ipv4":"5.6.7.8","region":"eu-west1"}]}}`,
		"clearEgressGateways": `{"data":{"egressGatewayAssociationsClear":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "a1a2a3a4-1111-2222-3333-444455556666"
  environment_id = "b1b2b3b4-5555-6666-7777-888899990000"
  region         = "eu-west1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "id", "a1a2a3a4-1111-2222-3333-444455556666:b1b2b3b4-5555-6666-7777-888899990000"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "service_id", "a1a2a3a4-1111-2222-3333-444455556666"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "environment_id", "b1b2b3b4-5555-6666-7777-888899990000"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "region", "eu-west1"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "ip_addresses.#", "1"),
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "ip_addresses.0", "5.6.7.8"),
				),
			},
		},
	})
}

func TestEgressGatewayResource_importState(t *testing.T) {
	server := newMockGraphQLServer(t, mockFixtures{
		"createEgressGateway": `{"data":{"egressGatewayAssociationCreate":[{"ipv4":"10.0.0.1","region":"us-west1"}]}}`,
		"getEgressGateways":   `{"data":{"egressGateways":[{"ipv4":"10.0.0.1","region":"us-west1"}]}}`,
		"clearEgressGateways": `{"data":{"egressGatewayAssociationsClear":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create the resource first
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "d1d2d3d4-aaaa-bbbb-cccc-ddddeeee0001"
  environment_id = "e1e2e3e4-aaaa-bbbb-cccc-ddddeeee0002"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_egress_gateway.test", "id", "d1d2d3d4-aaaa-bbbb-cccc-ddddeeee0001:e1e2e3e4-aaaa-bbbb-cccc-ddddeeee0002"),
				),
			},
			// Import using service_id:environment_id format
			{
				ResourceName:            "railway_egress_gateway.test",
				ImportState:             true,
				ImportStateId:           "d1d2d3d4-aaaa-bbbb-cccc-ddddeeee0001:e1e2e3e4-aaaa-bbbb-cccc-ddddeeee0002",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"region"},
			},
		},
	})
}

func TestEgressGatewayImportParsing(t *testing.T) {
	server := newMockGraphQLServer(t, mockFixtures{
		"createEgressGateway": `{"data":{"egressGatewayAssociationCreate":[{"ipv4":"10.0.0.1","region":"us-west1"}]}}`,
		"getEgressGateways":   `{"data":{"egressGateways":[{"ipv4":"10.0.0.1","region":"us-west1"}]}}`,
		"clearEgressGateways": `{"data":{"egressGatewayAssociationsClear":true}}`,
	})
	defer server.Close()

	tests := []struct {
		name        string
		importId    string
		expectError string
	}{
		{
			name:        "too few parts",
			importId:    "only-one-part",
			expectError: "Expected import identifier with format: service_id:environment_id",
		},
		{
			name:        "too many parts",
			importId:    "part1:part2:part3",
			expectError: "Expected import identifier with format: service_id:environment_id",
		},
		{
			name:        "empty service_id",
			importId:    ":env-id",
			expectError: "Expected import identifier with format: service_id:environment_id",
		},
		{
			name:        "empty environment_id",
			importId:    "svc-id:",
			expectError: "Expected import identifier with format: service_id:environment_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					// Create a resource to have something in state
					{
						Config: testUnitProviderConfig(server.URL) + `
resource "railway_egress_gateway" "test" {
  service_id     = "d1d2d3d4-aaaa-bbbb-cccc-ddddeeee0001"
  environment_id = "e1e2e3e4-aaaa-bbbb-cccc-ddddeeee0002"
}`,
					},
					// Attempt import with invalid ID
					{
						ResourceName:  "railway_egress_gateway.test",
						ImportState:   true,
						ImportStateId: tt.importId,
						ExpectError:   regexp.MustCompile(tt.expectError),
					},
				},
			})
		})
	}
}

