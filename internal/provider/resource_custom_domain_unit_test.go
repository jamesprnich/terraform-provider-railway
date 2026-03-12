package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCustomDomainResource_targetPortSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaResp := &fwresource.SchemaResponse{}
	NewCustomDomainResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	attr, ok := schemaResp.Schema.Attributes["target_port"]
	if !ok {
		t.Fatal("target_port attribute not found in schema")
	}

	int64Attr, ok := attr.(fwschema.Int64Attribute)
	if !ok {
		t.Fatal("target_port attribute is not Int64Attribute")
	}

	if !int64Attr.Optional {
		t.Error("target_port should be Optional")
	}
	if !int64Attr.Computed {
		t.Error("target_port should be Computed")
	}
}

func TestCustomDomainResource_withTargetPort(t *testing.T) {
	server := newMockGraphQLServer(t, mockFixtures{
		"getService":         `{"data":{"service":{"id":"00000000-0000-0000-0000-000000000003","name":"api","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
		"createCustomDomain": `{"data":{"customDomainCreate":{"id":"cd-123","domain":"app.example.com","targetPort":8080,"status":{"dnsRecords":[{"hostlabel":"app","requiredValue":"cname.railway.app","zone":"example.com"}]},"environmentId":"00000000-0000-0000-0000-000000000002","serviceId":"00000000-0000-0000-0000-000000000003"}}}`,
		"listCustomDomains":  `{"data":{"domains":{"customDomains":[{"id":"cd-123","domain":"app.example.com","targetPort":8080,"status":{"dnsRecords":[{"hostlabel":"app","requiredValue":"cname.railway.app","zone":"example.com"}]},"environmentId":"00000000-0000-0000-0000-000000000002","serviceId":"00000000-0000-0000-0000-000000000003"}]}}}`,
		"deleteCustomDomain": `{"data":{"customDomainDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "app.example.com"
  environment_id = "00000000-0000-0000-0000-000000000002"
  service_id     = "00000000-0000-0000-0000-000000000003"
  target_port    = 8080
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_custom_domain.test", "id", "cd-123"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "domain", "app.example.com"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "target_port", "8080"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "app"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "zone", "example.com"),
				),
			},
		},
	})
}

func TestCustomDomainResource_withoutTargetPort(t *testing.T) {
	server := newMockGraphQLServer(t, mockFixtures{
		"getService":         `{"data":{"service":{"id":"00000000-0000-0000-0000-000000000003","name":"api","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
		"createCustomDomain": `{"data":{"customDomainCreate":{"id":"cd-456","domain":"web.example.com","targetPort":0,"status":{"dnsRecords":[{"hostlabel":"web","requiredValue":"cname.railway.app","zone":"example.com"}]},"environmentId":"00000000-0000-0000-0000-000000000002","serviceId":"00000000-0000-0000-0000-000000000003"}}}`,
		"listCustomDomains":  `{"data":{"domains":{"customDomains":[{"id":"cd-456","domain":"web.example.com","targetPort":0,"status":{"dnsRecords":[{"hostlabel":"web","requiredValue":"cname.railway.app","zone":"example.com"}]},"environmentId":"00000000-0000-0000-0000-000000000002","serviceId":"00000000-0000-0000-0000-000000000003"}]}}}`,
		"deleteCustomDomain": `{"data":{"customDomainDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "web.example.com"
  environment_id = "00000000-0000-0000-0000-000000000002"
  service_id     = "00000000-0000-0000-0000-000000000003"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_custom_domain.test", "id", "cd-456"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "domain", "web.example.com"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "web"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "zone", "example.com"),
					resource.TestCheckNoResourceAttr("railway_custom_domain.test", "target_port"),
				),
			},
		},
	})
}
