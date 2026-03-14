package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestServiceInstanceResource_update(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	v1Response := `{"data":{"serviceInstance":{"id":"si-upd","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"npm run build","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":"us-west1","railwayConfigFile":"","cronSchedule":"","sleepApplication":false}}}`
	v2Response := `{"data":{"serviceInstance":{"id":"si-upd","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"node server.js","buildCommand":"npm ci","rootDirectory":"","healthcheckPath":"/health","numReplicas":2,"region":"us-west1","railwayConfigFile":"","cronSchedule":"","sleepApplication":false}}}`

	var mu sync.Mutex
	updateCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock server: failed to decode request body: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		mu.Lock()
		defer mu.Unlock()

		switch req.OperationName {
		case "getServiceInstanceDetailed":
			// After 2nd update (the actual update in step 2), return v2 response
			if updateCount >= 2 {
				fmt.Fprint(w, v2Response)
			} else {
				fmt.Fprint(w, v1Response)
			}
		case "updateServiceInstanceInEnvironment":
			updateCount++
			fmt.Fprint(w, `{"data":{"serviceInstanceUpdate":true}}`)
		case "deployServiceInstance":
			fmt.Fprint(w, `{"data":{"serviceInstanceDeploy":true}}`)
		case "redeployServiceInstance":
			fmt.Fprint(w, `{"data":{"serviceInstanceRedeploy":true}}`)
		default:
			t.Errorf("mock server: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "` + serviceId + `"
  environment_id = "` + envId + `"
  start_command  = "npm start"
  build_command  = "npm run build"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "start_command", "npm start"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "build_command", "npm run build"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "num_replicas", "1"),
				),
			},
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id      = "` + serviceId + `"
  environment_id  = "` + envId + `"
  start_command   = "node server.js"
  build_command   = "npm ci"
  healthcheck_path = "/health"
  num_replicas    = 2
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "start_command", "node server.js"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "build_command", "npm ci"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", "/health"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "num_replicas", "2"),
				),
			},
		},
	})
}

func TestServiceInstanceResource_disappears(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"getServiceInstanceDetailed":         `{"data":{"serviceInstance":{"id":"si-dis","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":"us-west1","railwayConfigFile":"","cronSchedule":"","sleepApplication":false}}}`,
		"updateServiceInstanceInEnvironment": `{"data":{"serviceInstanceUpdate":true}}`,
		"deployServiceInstance":              `{"data":{"serviceInstanceDeploy":true}}`,
	}, "getServiceInstanceDetailed")
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
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "service_id", serviceId),
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
