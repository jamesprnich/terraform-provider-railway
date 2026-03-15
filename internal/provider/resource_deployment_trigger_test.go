package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDeploymentTriggerResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentTriggerResourceConfig("main"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "jamesprnich/terraform-provider-railway"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "main"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "source_provider", "github"),
				),
			},
			// Import
			{
				ResourceName: "railway_deployment_trigger.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["railway_deployment_trigger.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return testAccProjectId + ":" + testAccEnvironmentId + ":" + testAccServiceId + ":" + rs.Primary.ID, nil
				},
				ImportStateVerify: true,
			},
			// Update branch
			{
				Config: testAccDeploymentTriggerResourceConfig("develop"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "develop"),
				),
			},
		},
	})
}

func TestAccDeploymentTriggerResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDeploymentTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentTriggerResourceConfig("main"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
					testAccCheckDeploymentTriggerDisappears("railway_deployment_trigger.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDeploymentTriggerResourceConfig(branch string) string {
	return fmt.Sprintf(`
resource "railway_deployment_trigger" "test" {
  service_id      = "%s"
  environment_id  = "%s"
  project_id      = "%s"
  repository      = "jamesprnich/terraform-provider-railway"
  branch          = "%s"
  source_provider = "github"
}
`, testAccServiceId, testAccEnvironmentId, testAccProjectId, branch)
}

func TestDeploymentTriggerResource_basic(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{
		"createDeploymentTrigger": `{"data":{"deploymentTriggerCreate":{"id":"dt-123","branch":"main","checkSuites":false,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/repo","serviceId":"00000000-0000-0000-0000-000000000003"}}}`,
		"getDeploymentTriggers":  `{"data":{"deploymentTriggers":{"edges":[{"node":{"id":"dt-123","branch":"main","checkSuites":false,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/repo","serviceId":"00000000-0000-0000-0000-000000000003"}}]}}}`,
		"deleteDeploymentTrigger": `{"data":{"deploymentTriggerDelete":true}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_deployment_trigger" "test" {
  service_id     = "00000000-0000-0000-0000-000000000003"
  environment_id = "00000000-0000-0000-0000-000000000002"
  project_id     = "00000000-0000-0000-0000-000000000001"
  repository     = "owner/repo"
  branch         = "main"
  source_provider = "github"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "id", "dt-123"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "main"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "owner/repo"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "source_provider", "github"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "false"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "00000000-0000-0000-0000-000000000003"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "00000000-0000-0000-0000-000000000002"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "00000000-0000-0000-0000-000000000001"),
				),
			},
		},
	})
}

func TestDeploymentTriggerResource_withOptionalFields(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{
		"createDeploymentTrigger": `{"data":{"deploymentTriggerCreate":{"id":"dt-456","branch":"develop","checkSuites":true,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/monorepo","serviceId":"00000000-0000-0000-0000-000000000003"}}}`,
		"getDeploymentTriggers":  `{"data":{"deploymentTriggers":{"edges":[{"node":{"id":"dt-456","branch":"develop","checkSuites":true,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/monorepo","serviceId":"00000000-0000-0000-0000-000000000003"}}]}}}`,
		"deleteDeploymentTrigger": `{"data":{"deploymentTriggerDelete":true}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_deployment_trigger" "test" {
  service_id     = "00000000-0000-0000-0000-000000000003"
  environment_id = "00000000-0000-0000-0000-000000000002"
  project_id     = "00000000-0000-0000-0000-000000000001"
  repository     = "owner/monorepo"
  branch         = "develop"
  source_provider = "github"
  check_suites   = true
  root_directory = "packages/api"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "id", "dt-456"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "develop"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "owner/monorepo"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "source_provider", "github"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "true"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "root_directory", "packages/api"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "00000000-0000-0000-0000-000000000003"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "00000000-0000-0000-0000-000000000002"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "00000000-0000-0000-0000-000000000001"),
				),
			},
		},
	})
}

func TestDeploymentTriggerResource_disappears(t *testing.T) {
	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createDeploymentTrigger":  `{"data":{"deploymentTriggerCreate":{"id":"dt-dis","branch":"main","checkSuites":false,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/repo","serviceId":"00000000-0000-0000-0000-000000000003"}}}`,
		"getDeploymentTriggers":    `{"data":{"deploymentTriggers":{"edges":[{"node":{"id":"dt-dis","branch":"main","checkSuites":false,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/repo","serviceId":"00000000-0000-0000-0000-000000000003"}}]}}}`,
		"deleteDeploymentTrigger":  `{"data":{"deploymentTriggerDelete":true}}`,
	}, "getDeploymentTriggers")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_deployment_trigger" "test" {
  service_id      = "00000000-0000-0000-0000-000000000003"
  environment_id  = "00000000-0000-0000-0000-000000000002"
  project_id      = "00000000-0000-0000-0000-000000000001"
  repository      = "owner/repo"
  branch          = "main"
  source_provider = "github"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "id", "dt-dis"),
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

func TestDeploymentTriggerResource_import(t *testing.T) {
	srv := newMockGraphQLServer(t, mockFixtures{
		"createDeploymentTrigger": `{"data":{"deploymentTriggerCreate":{"id":"dt-789","branch":"main","checkSuites":false,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/repo","serviceId":"00000000-0000-0000-0000-000000000003"}}}`,
		"getDeploymentTriggers":  `{"data":{"deploymentTriggers":{"edges":[{"node":{"id":"dt-789","branch":"main","checkSuites":false,"environmentId":"00000000-0000-0000-0000-000000000002","projectId":"00000000-0000-0000-0000-000000000001","provider":"github","repository":"owner/repo","serviceId":"00000000-0000-0000-0000-000000000003"}}]}}}`,
		"deleteDeploymentTrigger": `{"data":{"deploymentTriggerDelete":true}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_deployment_trigger" "test" {
  service_id     = "00000000-0000-0000-0000-000000000003"
  environment_id = "00000000-0000-0000-0000-000000000002"
  project_id     = "00000000-0000-0000-0000-000000000001"
  repository     = "owner/repo"
  branch         = "main"
  source_provider = "github"
}
`,
			},
			{
				ResourceName:      "railway_deployment_trigger.test",
				ImportState:       true,
				ImportStateId:     "00000000-0000-0000-0000-000000000001:00000000-0000-0000-0000-000000000002:00000000-0000-0000-0000-000000000003:dt-789",
				ImportStateVerify: true,
			},
		},
	})
}
