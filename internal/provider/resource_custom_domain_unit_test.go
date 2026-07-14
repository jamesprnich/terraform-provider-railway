package provider

import (
	"context"
	"strconv"
	"strings"
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

// customDomainFixture builds a customDomain JSON blob with a realistic
// two-record dnsRecords shape (CNAME + TXT) plus the verification triple.
// Parameterised on the record ordering so tests can assert the type-based
// selector picks the CNAME regardless of array position — that ordering
// invariance is the specific v0.11.4 fix.
func customDomainFixture(id, domain string, targetPort int, txtFirst bool) string {
	cnameRecord := `{"recordType":"DNS_RECORD_TYPE_CNAME","purpose":"DNS_RECORD_PURPOSE_TRAFFIC_ROUTE","hostlabel":"app","requiredValue":"cname.railway.app","zone":"example.com"}`
	txtRecord := `{"recordType":"DNS_RECORD_TYPE_TXT","purpose":"DNS_RECORD_PURPOSE_ACME_DNS01_CHALLENGE","hostlabel":"_railway-verify.app","requiredValue":"eyJhbGciOiJIUzI1NiJ9-verification-token","zone":"example.com"}`
	records := cnameRecord + "," + txtRecord
	if txtFirst {
		records = txtRecord + "," + cnameRecord
	}
	status := `"status":{"verified":false,"verificationDnsHost":"_railway-verify.app.example.com","verificationToken":"eyJhbGciOiJIUzI1NiJ9-verification-token","dnsRecords":[` + records + `]}`
	return `{"id":"` + id + `","domain":"` + domain + `","targetPort":` + strconv.Itoa(targetPort) + `,` + status + `,"environmentId":"00000000-0000-0000-0000-000000000002","serviceId":"00000000-0000-0000-0000-000000000003"}`
}

func TestCustomDomainResource_withTargetPort(t *testing.T) {
	cd := customDomainFixture("cd-123", "app.example.com", 8080, false)
	server := newMockGraphQLServer(t, mockFixtures{
		"getService":         `{"data":{"service":{"id":"00000000-0000-0000-0000-000000000003","name":"api","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
		"createCustomDomain": `{"data":{"customDomainCreate":` + cd + `}}`,
		"listCustomDomains":  `{"data":{"domains":{"customDomains":[` + cd + `]}}}`,
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
					resource.TestCheckResourceAttr("railway_custom_domain.test", "dns_record_value", "cname.railway.app"),
				),
			},
		},
	})
}

func TestCustomDomainResource_withoutTargetPort(t *testing.T) {
	cd := customDomainFixture("cd-456", "web.example.com", 0, false)
	// Re-map hostlabel/zone to match "web" to keep the existing test's assertions realistic.
	cd = strings.Replace(cd, `"hostlabel":"app"`, `"hostlabel":"web"`, 1)
	cd = strings.Replace(cd, `"hostlabel":"_railway-verify.app"`, `"hostlabel":"_railway-verify.web"`, 1)
	cd = strings.Replace(cd, `"verificationDnsHost":"_railway-verify.app.example.com"`, `"verificationDnsHost":"_railway-verify.web.example.com"`, 1)

	server := newMockGraphQLServer(t, mockFixtures{
		"getService":         `{"data":{"service":{"id":"00000000-0000-0000-0000-000000000003","name":"api","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
		"createCustomDomain": `{"data":{"customDomainCreate":` + cd + `}}`,
		"listCustomDomains":  `{"data":{"domains":{"customDomains":[` + cd + `]}}}`,
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

// TestCustomDomainResource_cnameSelectionIsOrderIndependent is the direct
// regression test for the pre-v0.11.4 bug. Railway returns dnsRecords as an
// unordered array containing at least the CNAME (for traffic routing) and a
// TXT (for domain verification). The previous provider took dnsRecords[0]
// unconditionally, which happened to be correct when Railway returned the
// CNAME first but would silently substitute the TXT's `requiredValue` (the
// verification token) for `dns_record_value` on any reorder — producing a
// CNAME pointing at a JWT string. This test drives Read against both
// orderings and asserts dns_record_value is the CNAME target in both.
//
// A test with a single-element dnsRecords array cannot detect this — index-0
// selection passes it trivially.
func TestCustomDomainResource_cnameSelectionIsOrderIndependent(t *testing.T) {
	for _, orderTxtFirst := range []bool{false, true} {
		orderName := "CNAME-first"
		if orderTxtFirst {
			orderName = "TXT-first"
		}
		t.Run(orderName, func(t *testing.T) {
			cd := customDomainFixture("cd-order", "app.example.com", 0, orderTxtFirst)
			server := newMockGraphQLServer(t, mockFixtures{
				"getService":         `{"data":{"service":{"id":"00000000-0000-0000-0000-000000000003","name":"api","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
				"createCustomDomain": `{"data":{"customDomainCreate":` + cd + `}}`,
				"listCustomDomains":  `{"data":{"domains":{"customDomains":[` + cd + `]}}}`,
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
}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							// The CNAME's own values, never the TXT's, regardless of
							// which record appeared first in the array.
							resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "app"),
							resource.TestCheckResourceAttr("railway_custom_domain.test", "zone", "example.com"),
							resource.TestCheckResourceAttr("railway_custom_domain.test", "dns_record_value", "cname.railway.app"),
							// And the verification fields carry the TXT's info via
							// Railway's dedicated status.verificationDnsHost/Token, not
							// via the dnsRecords array — order-independent by construction.
							resource.TestCheckResourceAttr("railway_custom_domain.test", "verified", "false"),
							resource.TestCheckResourceAttr("railway_custom_domain.test", "verification_dns_host", "_railway-verify.app.example.com"),
							resource.TestCheckResourceAttr("railway_custom_domain.test", "verification_token", "eyJhbGciOiJIUzI1NiJ9-verification-token"),
						),
					},
				},
			})
		})
	}
}

// TestCustomDomainResource_cnameFallsBackToRecordType covers the case where
// Railway returns records with purpose=UNSPECIFIED (older API responses or
// edge cases). Selection should fall back to recordType == CNAME and still
// pick the correct record.
func TestCustomDomainResource_cnameFallsBackToRecordType(t *testing.T) {
	cd := `{"id":"cd-fallback","domain":"legacy.example.com","targetPort":0,"status":{"verified":true,"verificationDnsHost":"","verificationToken":"","dnsRecords":[` +
		`{"recordType":"DNS_RECORD_TYPE_TXT","purpose":"DNS_RECORD_PURPOSE_UNSPECIFIED","hostlabel":"_railway-verify.legacy","requiredValue":"txt-value","zone":"example.com"},` +
		`{"recordType":"DNS_RECORD_TYPE_CNAME","purpose":"DNS_RECORD_PURPOSE_UNSPECIFIED","hostlabel":"legacy","requiredValue":"cname.railway.app","zone":"example.com"}` +
		`]},"environmentId":"00000000-0000-0000-0000-000000000002","serviceId":"00000000-0000-0000-0000-000000000003"}`

	server := newMockGraphQLServer(t, mockFixtures{
		"getService":         `{"data":{"service":{"id":"00000000-0000-0000-0000-000000000003","name":"api","projectId":"00000000-0000-0000-0000-000000000001"}}}`,
		"createCustomDomain": `{"data":{"customDomainCreate":` + cd + `}}`,
		"listCustomDomains":  `{"data":{"domains":{"customDomains":[` + cd + `]}}}`,
		"deleteCustomDomain": `{"data":{"customDomainDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_custom_domain" "test" {
  domain         = "legacy.example.com"
  environment_id = "00000000-0000-0000-0000-000000000002"
  service_id     = "00000000-0000-0000-0000-000000000003"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Even though the TXT is first in the array and no purpose is set,
					// the recordType fallback finds the CNAME correctly.
					resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "legacy"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "dns_record_value", "cname.railway.app"),
					// Empty verification fields become null, not "" — the provider
					// distinguishes "verified" from "empty verification info".
					resource.TestCheckResourceAttr("railway_custom_domain.test", "verified", "true"),
					resource.TestCheckNoResourceAttr("railway_custom_domain.test", "verification_dns_host"),
					resource.TestCheckNoResourceAttr("railway_custom_domain.test", "verification_token"),
				),
			},
		},
	})
}

// TestSelectTrafficRouteCNAME_pure hits the selector function directly. Faster
// than the framework-driven tests above and easier to add cases to as we
// discover new Railway response shapes.
func TestSelectTrafficRouteCNAME_pure(t *testing.T) {
	t.Parallel()

	rec := func(rt DNSRecordType, purpose DNSRecordPurpose, host string) CustomDomainStatusDnsRecordsDNSRecords {
		return CustomDomainStatusDnsRecordsDNSRecords{
			RecordType:    rt,
			Purpose:       purpose,
			Hostlabel:     host,
			RequiredValue: host + "-value",
			Zone:          "example.com",
		}
	}

	tests := []struct {
		name      string
		records   []CustomDomainStatusDnsRecordsDNSRecords
		wantNil   bool
		wantLabel string
	}{
		{
			name:    "empty array returns nil (Railway hasn't computed status yet)",
			records: nil,
			wantNil: true,
		},
		{
			name: "CNAME with TRAFFIC_ROUTE purpose is selected over TXT",
			records: []CustomDomainStatusDnsRecordsDNSRecords{
				rec(DNSRecordTypeDnsRecordTypeCname, DNSRecordPurposeDnsRecordPurposeTrafficRoute, "app"),
				rec(DNSRecordTypeDnsRecordTypeTxt, DNSRecordPurposeDnsRecordPurposeAcmeDns01Challenge, "_railway-verify.app"),
			},
			wantLabel: "app",
		},
		{
			name: "reversed order still picks CNAME — regression for the pre-v0.11.4 [0] bug",
			records: []CustomDomainStatusDnsRecordsDNSRecords{
				rec(DNSRecordTypeDnsRecordTypeTxt, DNSRecordPurposeDnsRecordPurposeAcmeDns01Challenge, "_railway-verify.app"),
				rec(DNSRecordTypeDnsRecordTypeCname, DNSRecordPurposeDnsRecordPurposeTrafficRoute, "app"),
			},
			wantLabel: "app",
		},
		{
			name: "when no purpose is set (UNSPECIFIED), falls back to recordType",
			records: []CustomDomainStatusDnsRecordsDNSRecords{
				rec(DNSRecordTypeDnsRecordTypeTxt, DNSRecordPurposeDnsRecordPurposeUnspecified, "_railway-verify.app"),
				rec(DNSRecordTypeDnsRecordTypeCname, DNSRecordPurposeDnsRecordPurposeUnspecified, "app"),
			},
			wantLabel: "app",
		},
		{
			name: "only-TXT array returns nil (no CNAME present at all)",
			records: []CustomDomainStatusDnsRecordsDNSRecords{
				rec(DNSRecordTypeDnsRecordTypeTxt, DNSRecordPurposeDnsRecordPurposeAcmeDns01Challenge, "_railway-verify.app"),
			},
			wantNil: true,
		},
		{
			name: "purpose takes precedence over recordType — a mislabeled TXT with TRAFFIC_ROUTE purpose (defensive; shouldn't happen in prod)",
			records: []CustomDomainStatusDnsRecordsDNSRecords{
				rec(DNSRecordTypeDnsRecordTypeCname, DNSRecordPurposeDnsRecordPurposeAcmeDns01Challenge, "cname-mislabeled"),
				rec(DNSRecordTypeDnsRecordTypeTxt, DNSRecordPurposeDnsRecordPurposeTrafficRoute, "txt-mislabeled"),
			},
			wantLabel: "txt-mislabeled",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := selectTrafficRouteCNAME(tc.records)
			if tc.wantNil {
				if got != nil {
					t.Errorf("selectTrafficRouteCNAME() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("selectTrafficRouteCNAME() = nil, want record with hostlabel=%q", tc.wantLabel)
			}
			if got.Hostlabel != tc.wantLabel {
				t.Errorf("selectTrafficRouteCNAME() picked hostlabel=%q, want %q", got.Hostlabel, tc.wantLabel)
			}
		})
	}
}
