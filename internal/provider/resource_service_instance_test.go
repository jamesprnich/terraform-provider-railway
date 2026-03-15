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

func TestAccServiceInstanceResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServiceInstanceDestroy,
		Steps: []resource.TestStep{
			// Create with start command
			{
				Config: testAccServiceInstanceResourceConfig("echo hello", "", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_service_instance.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_service_instance.test", "start_command", "echo hello"),
				),
			},
			// Import
			{
				ResourceName:      "railway_service_instance.test",
				ImportState:       true,
				ImportStateId:     testAccServiceId + ":" + testAccEnvironmentId,
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
					"vcpus",
					"memory_gb",
					"sleep_application",
					"overlap_seconds",
					"draining_seconds",
					"healthcheck_timeout",
					"restart_policy_type",
					"restart_policy_max_retries",
					"pre_deploy_command",
					"watch_patterns",
					"builder",
				},
			},
			// Update
			{
				Config: testAccServiceInstanceResourceConfig("echo updated", "", "/health"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "start_command", "echo updated"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", "/health"),
				),
			},
		},
	})
}

func testAccServiceInstanceResourceConfig(startCmd, buildCmd, healthcheckPath string) string {
	config := fmt.Sprintf(`
resource "railway_service_instance" "test" {
  service_id     = "%s"
  environment_id = "%s"
`, testAccServiceId, testAccEnvironmentId)
	if startCmd != "" {
		config += fmt.Sprintf(`  start_command  = "%s"
`, startCmd)
	}
	if buildCmd != "" {
		config += fmt.Sprintf(`  build_command  = "%s"
`, buildCmd)
	}
	if healthcheckPath != "" {
		config += fmt.Sprintf(`  healthcheck_path = "%s"
`, healthcheckPath)
	}
	config += "}\n"
	return config
}

func TestServiceInstanceResource_basic(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	instanceResp := `{"data":{"serviceInstance":{"id":"si-123","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"npm run build","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}`

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

	v1Response := `{"data":{"serviceInstance":{"id":"si-upd","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"npm run build","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}`
	v2Response := `{"data":{"serviceInstance":{"id":"si-upd","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"node server.js","buildCommand":"npm ci","rootDirectory":"","healthcheckPath":"/health","numReplicas":2,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":2}}}}}}}}}`

	var mu sync.Mutex
	updateCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
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
		"getServiceInstanceDetailed":         `{"data":{"serviceInstance":{"id":"si-dis","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}`,
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

	instanceResp := `{"data":{"serviceInstance":{"id":"si-456","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"gunicorn app:app","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":2,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":2}}}}}}}}}`

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
					"sleep_application",
					"overlap_seconds",
					"draining_seconds",
					"healthcheck_timeout",
					"restart_policy_type",
					"restart_policy_max_retries",
					"pre_deploy_command",
					"watch_patterns",
					"builder",
				},
			},
		},
	})
}

func TestServiceInstanceResource_regionChange(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	v1Response := `{"data":{"serviceInstance":{"id":"si-region","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}`
	v2Response := `{"data":{"serviceInstance":{"id":"si-region","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":2,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"asia-southeast1":{"numReplicas":2}}}}}}}}}`

	var mu sync.Mutex
	updateCount := 0
	var lastUpdateInput map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
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
			if updateCount >= 2 {
				fmt.Fprint(w, v2Response)
			} else {
				fmt.Fprint(w, v1Response)
			}
		case "updateServiceInstanceInEnvironment":
			updateCount++
			// Capture the input to verify multiRegionConfig is used
			var variables map[string]interface{}
			if err := json.Unmarshal(req.Variables, &variables); err == nil {
				if input, ok := variables["input"].(map[string]interface{}); ok {
					lastUpdateInput = input
				}
			}
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
			// Step 1: Create with us-west1
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "` + serviceId + `"
  environment_id = "` + envId + `"
  start_command  = "npm start"
  region         = "us-west1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "region", "us-west1"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "num_replicas", "1"),
					func(s *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						// Verify multiRegionConfig was sent (not the top-level region field)
						if lastUpdateInput == nil {
							return fmt.Errorf("expected update input to be captured")
						}
						if _, hasMultiRegion := lastUpdateInput["multiRegionConfig"]; !hasMultiRegion {
							return fmt.Errorf("expected multiRegionConfig in update input, got: %v", lastUpdateInput)
						}
						if _, hasRegion := lastUpdateInput["region"]; hasRegion {
							return fmt.Errorf("expected no top-level region field, but found one in: %v", lastUpdateInput)
						}
						return nil
					},
				),
			},
			// Step 2: Change to asia-southeast1 with 2 replicas
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id     = "` + serviceId + `"
  environment_id = "` + envId + `"
  start_command  = "npm start"
  region         = "asia-southeast1"
  num_replicas   = 2
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "region", "asia-southeast1"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "num_replicas", "2"),
					func(s *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if lastUpdateInput == nil {
							return fmt.Errorf("expected update input to be captured")
						}
						if _, hasMultiRegion := lastUpdateInput["multiRegionConfig"]; !hasMultiRegion {
							return fmt.Errorf("expected multiRegionConfig in update input for region change, got: %v", lastUpdateInput)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestServiceInstanceResource_deploySettings(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	instanceResp := `{"data":{"serviceInstance":{"id":"si-deploy","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":2,"drainingSeconds":3,"healthcheckTimeout":300,"restartPolicyType":"ON_FAILURE","restartPolicyMaxRetries":5,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":["server/**"],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}`

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
  service_id              = "` + serviceId + `"
  environment_id          = "` + envId + `"
  start_command           = "npm start"
  overlap_seconds         = 2
  draining_seconds        = 3
  healthcheck_timeout     = 300
  restart_policy_type     = "ON_FAILURE"
  restart_policy_max_retries = 5
  builder                 = "RAILPACK"
  watch_patterns          = ["server/**"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "service_id", serviceId),
					resource.TestCheckResourceAttr("railway_service_instance.test", "environment_id", envId),
					resource.TestCheckResourceAttr("railway_service_instance.test", "start_command", "npm start"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "overlap_seconds", "2"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "draining_seconds", "3"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_timeout", "300"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "restart_policy_max_retries", "5"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "builder", "RAILPACK"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "watch_patterns.#", "1"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "watch_patterns.0", "server/**"),
				),
			},
		},
	})
}
