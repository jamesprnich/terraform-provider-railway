package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServiceInstanceResource_basic(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	instanceResp := `{"data":{"serviceInstance":{"id":"si-123","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"npm run build","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":"us-west1","railwayConfigFile":"","cronSchedule":"","sleepApplication":false}}}`

	srv := newMockGraphQLServer(t, mockFixtures{
		"getServiceInstanceDetailed":         instanceResp,
		"updateServiceInstanceInEnvironment": `{"data":{"serviceInstanceUpdate":true}}`,
		"deployServiceInstance":              `{"data":{"serviceInstanceDeploy":true}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "` + serviceId + `"
  environment_id = "` + envId + `"
  start_command  = "npm start"
  build_command  = "npm run build"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "service_id", serviceId),
					resource.TestCheckResourceAttr("railway_service_instance.test", "environment_id", envId),
					resource.TestCheckResourceAttr("railway_service_instance.test", "start_command", "npm start"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "num_replicas", "1"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "region", "us-west1"),
				),
			},
		},
	})
}

func TestServiceInstanceResource_import(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	instanceResp := `{"data":{"serviceInstance":{"id":"si-456","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"gunicorn app:app","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":2,"region":"us-west1","railwayConfigFile":"","cronSchedule":"","sleepApplication":false}}}`

	srv := newMockGraphQLServer(t, mockFixtures{
		"getServiceInstanceDetailed":         instanceResp,
		"updateServiceInstanceInEnvironment": `{"data":{"serviceInstanceUpdate":true}}`,
		"deployServiceInstance":              `{"data":{"serviceInstanceDeploy":true}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "` + serviceId + `"
  environment_id = "` + envId + `"
  num_replicas   = 2
}
`,
			},
			{
				ResourceName:  "railway_service_instance.test",
				ImportState:   true,
				ImportStateId: serviceId + ":" + envId,
				// Optional-only fields (start_command, build_command, root_directory, etc.)
				// are intentionally not adopted from API on import — they may be set at the
				// service level and shouldn't be pulled into service_instance state.
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"start_command",
					"build_command",
					"root_directory",
					"healthcheck_path",
					"config_path",
					"source_image",
					"source_repo",
					"cron_schedule",
				},
			},
		},
	})
}
