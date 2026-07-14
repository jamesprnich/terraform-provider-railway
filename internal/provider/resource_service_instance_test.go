package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
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
					"registry_credentials",
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
				_, _ = fmt.Fprint(w, v2Response)
			} else {
				_, _ = fmt.Fprint(w, v1Response)
			}
		case "updateServiceInstanceInEnvironment":
			updateCount++
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceUpdate":true}}`)
		case "deployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceDeploy":true}}`)
		case "redeployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceRedeploy":true}}`)
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
				_, _ = fmt.Fprint(w, v2Response)
			} else {
				_, _ = fmt.Fprint(w, v1Response)
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
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceUpdate":true}}`)
		case "deployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceDeploy":true}}`)
		case "redeployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceRedeploy":true}}`)
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

func TestServiceInstanceResource_registryCredentials(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	// API response does not include registryCredentials — it is write-only.
	instanceResp := `{"data":{"serviceInstance":{"id":"si-priv","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":null,"watchPatterns":[],"source":{"image":"ghcr.io/owner/app@sha256:abc123","repo":null},"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}` //nolint

	var mu sync.Mutex
	var capturedInput map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock server: failed to decode request body: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.OperationName {
		case "getServiceInstanceDetailed":
			_, _ = fmt.Fprint(w, instanceResp)
		case "updateServiceInstanceInEnvironment":
			mu.Lock()
			var variables map[string]interface{}
			if err := json.Unmarshal(req.Variables, &variables); err == nil {
				if input, ok := variables["input"].(map[string]interface{}); ok {
					capturedInput = input
				}
			}
			mu.Unlock()
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceUpdate":true}}`)
		case "deployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceDeploy":true}}`)
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
  source_image   = "ghcr.io/owner/app@sha256:abc123"
  registry_credentials = {
    username = "myuser"
    password = "ghp_secret-token"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "source_image", "ghcr.io/owner/app@sha256:abc123"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "registry_credentials.username", "myuser"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "registry_credentials.password", "ghp_secret-token"),
					func(s *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if capturedInput == nil {
							return fmt.Errorf("expected update input to be captured")
						}
						creds, ok := capturedInput["registryCredentials"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("expected registryCredentials in update input, got: %v", capturedInput)
						}
						if creds["username"] != "myuser" {
							return fmt.Errorf("expected username=myuser, got: %v", creds["username"])
						}
						if creds["password"] != "ghp_secret-token" {
							return fmt.Errorf("expected password=ghp_secret-token, got: %v", creds["password"])
						}
						return nil
					},
				),
			},
		},
	})
}

func TestServiceInstanceResource_registryCredentials_requiresSourceImage(t *testing.T) {
	t.Parallel()
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	server := newMockGraphQLServer(t, mockFixtures{})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id           = "` + serviceId + `"
  environment_id       = "` + envId + `"
  registry_credentials = {
    username = "myuser"
    password = "ghp_secret-token"
  }
}
`,
				ExpectError: regexp.MustCompile(`registry_credentials.*can only be set when.*source_image`),
			},
		},
	})
}

// Direct regression for the JSON→[]string unmarshal bug on
// ServiceInstance.preDeployCommand plus the shape-choice check. Railway's
// schema declares the field as `JSON` on the read type but the runtime value
// is a list of shell command strings — the previous *map[string]interface{}
// binding panicked on Read the moment the field was non-null. Ship-blocking:
// every refresh/plan hit this after the field was first set. This test also
// verifies the provider serialises the single-string HCL attribute as a
// one-element list on the wire, matching what Railway's server accepts.
func TestServiceInstanceResource_preDeployCommand_readSucceeds(t *testing.T) {
	t.Parallel()
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	// preDeployCommand comes back as an array on the wire — exactly the
	// shape that used to crash the JSON unmarshal.
	instanceResp := `{"data":{"serviceInstance":{"id":"si-pdc","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":["python manage.py migrate"],"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}` //nolint:lll

	var mu sync.Mutex
	var capturedInput map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mockGraphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock server: failed to decode request body: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.OperationName {
		case "getServiceInstanceDetailed":
			_, _ = fmt.Fprint(w, instanceResp)
		case "updateServiceInstanceInEnvironment":
			mu.Lock()
			var variables map[string]interface{}
			if err := json.Unmarshal(req.Variables, &variables); err == nil {
				if input, ok := variables["input"].(map[string]interface{}); ok {
					capturedInput = input
				}
			}
			mu.Unlock()
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceUpdate":true}}`)
		case "deployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceDeploy":true}}`)
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
  service_id         = "` + serviceId + `"
  environment_id     = "` + envId + `"
  start_command      = "npm start"
  pre_deploy_command = "python manage.py migrate"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "pre_deploy_command", "python manage.py migrate"),
					func(s *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if capturedInput == nil {
							return fmt.Errorf("expected update input to be captured")
						}
						raw, ok := capturedInput["preDeployCommand"]
						if !ok {
							return fmt.Errorf("expected preDeployCommand in update input, got: %v", capturedInput)
						}
						arr, ok := raw.([]interface{})
						if !ok {
							return fmt.Errorf("expected preDeployCommand to be serialised as an array on the wire, got %T: %v", raw, raw)
						}
						if len(arr) != 1 {
							return fmt.Errorf("expected exactly 1 command element on the wire (Railway rejects >1), got %d: %v", len(arr), arr)
						}
						if arr[0] != "python manage.py migrate" {
							return fmt.Errorf("expected command payload to match HCL string, got: %v", arr[0])
						}
						return nil
					},
				),
			},
		},
	})
}

// Full lifecycle regression for pre_deploy_command: Create → Read → Update → Read.
// The bug specifically fires on Read-after-Update because Read is what
// unmarshals the response body — a create-only test would not exercise the
// crash path. The wire type on both request and response is a list, but the
// user-facing HCL attribute is a single string; Railway's server-side
// validation only accepts a one-element list, so exercising an update from
// one string to another is the realistic user scenario.
func TestServiceInstanceResource_preDeployCommand_lifecycle(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	base := func(preDeploy string) string {
		return `{"data":{"serviceInstance":{"id":"si-pdcl","serviceId":"` + serviceId + `","environmentId":"` + envId + `","startCommand":"npm start","buildCommand":"","rootDirectory":"","healthcheckPath":"","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":null,"restartPolicyType":"ALWAYS","restartPolicyMaxRetries":0,"builder":"RAILPACK","preDeployCommand":` + preDeploy + `,"watchPatterns":[],"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}`
	}

	v1 := base(`["migrate"]`)
	v2 := base(`["seed"]`)

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
			if updateCount >= 2 {
				_, _ = fmt.Fprint(w, v2)
			} else {
				_, _ = fmt.Fprint(w, v1)
			}
		case "updateServiceInstanceInEnvironment":
			updateCount++
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceUpdate":true}}`)
		case "deployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceDeploy":true}}`)
		case "redeployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceRedeploy":true}}`)
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
  service_id         = "` + serviceId + `"
  environment_id     = "` + envId + `"
  start_command      = "npm start"
  pre_deploy_command = "migrate"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "pre_deploy_command", "migrate"),
				),
			},
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id         = "` + serviceId + `"
  environment_id     = "` + envId + `"
  start_command      = "npm start"
  pre_deploy_command = "seed"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "pre_deploy_command", "seed"),
				),
			},
		},
	})
}

// Comprehensive image-source lifecycle. Sets registry_credentials +
// restart_policy_* + healthcheck_* + pre_deploy_command together and drives
// Create → Read → Update → Read. Password is write-only (never in the API
// response) — proves Read preserves it from state so there is no perpetual
// diff. Also proves the full set of fields the reporter flagged do round-trip
// cleanly on an image-sourced service instance.
func TestServiceInstanceResource_imageSource_lifecycle(t *testing.T) {
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	build := func(startCmd, healthPath, restartType string, maxRetries int, hcTimeout int, preDeploy string) string {
		return fmt.Sprintf(`{"data":{"serviceInstance":{"id":"si-img","serviceId":"%s","environmentId":"%s","startCommand":"%s","buildCommand":"","rootDirectory":"","healthcheckPath":"%s","numReplicas":1,"region":null,"railwayConfigFile":"","cronSchedule":"","sleepApplication":false,"overlapSeconds":null,"drainingSeconds":null,"healthcheckTimeout":%d,"restartPolicyType":"%s","restartPolicyMaxRetries":%d,"builder":"RAILPACK","preDeployCommand":%s,"watchPatterns":[],"source":{"image":"ghcr.io/owner/app@sha256:abc123","repo":null},"latestDeployment":{"meta":{"serviceManifest":{"deploy":{"multiRegionConfig":{"us-west1":{"numReplicas":1}}}}}}}}}`,
			serviceId, envId, startCmd, healthPath, hcTimeout, restartType, maxRetries, preDeploy)
	}

	v1 := build("npm start", "/health", "ON_FAILURE", 3, 30, `["migrate"]`)
	v2 := build("node server.js", "/status", "ALWAYS", 5, 60, `["seed"]`)

	var mu sync.Mutex
	updateCount := 0
	var lastInput map[string]interface{}

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
				_, _ = fmt.Fprint(w, v2)
			} else {
				_, _ = fmt.Fprint(w, v1)
			}
		case "updateServiceInstanceInEnvironment":
			updateCount++
			var vars map[string]interface{}
			if err := json.Unmarshal(req.Variables, &vars); err == nil {
				if input, ok := vars["input"].(map[string]interface{}); ok {
					lastInput = input
				}
			}
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceUpdate":true}}`)
		case "deployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceDeploy":true}}`)
		case "redeployServiceInstance":
			_, _ = fmt.Fprint(w, `{"data":{"serviceInstanceRedeploy":true}}`)
		default:
			t.Errorf("mock server: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id                  = "` + serviceId + `"
  environment_id              = "` + envId + `"
  source_image                = "ghcr.io/owner/app@sha256:abc123"
  start_command               = "npm start"
  healthcheck_path            = "/health"
  healthcheck_timeout         = 30
  restart_policy_type         = "ON_FAILURE"
  restart_policy_max_retries  = 3
  pre_deploy_command          = "migrate"
  registry_credentials = {
    username = "myuser"
    password = "ghp_secret-token"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "source_image", "ghcr.io/owner/app@sha256:abc123"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", "/health"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_timeout", "30"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "restart_policy_max_retries", "3"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "pre_deploy_command", "migrate"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "registry_credentials.username", "myuser"),
					// Password is write-only — must be preserved from state on Read.
					resource.TestCheckResourceAttr("railway_service_instance.test", "registry_credentials.password", "ghp_secret-token"),
					func(s *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if lastInput == nil {
							return fmt.Errorf("expected create update input to be captured")
						}
						creds, ok := lastInput["registryCredentials"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("expected registryCredentials in create input, got: %v", lastInput)
						}
						if creds["password"] != "ghp_secret-token" {
							return fmt.Errorf("expected password sent on create, got: %v", creds["password"])
						}
						return nil
					},
				),
			},
			// Update — changes to healthcheck, restart policy, start command, and pre_deploy_command.
			// Credentials unchanged. Read-after-Update must not perpetual-diff on password.
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id                  = "` + serviceId + `"
  environment_id              = "` + envId + `"
  source_image                = "ghcr.io/owner/app@sha256:abc123"
  start_command               = "node server.js"
  healthcheck_path            = "/status"
  healthcheck_timeout         = 60
  restart_policy_type         = "ALWAYS"
  restart_policy_max_retries  = 5
  pre_deploy_command          = "seed"
  registry_credentials = {
    username = "myuser"
    password = "ghp_secret-token"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "start_command", "node server.js"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", "/status"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_timeout", "60"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "restart_policy_type", "ALWAYS"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "restart_policy_max_retries", "5"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "pre_deploy_command", "seed"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "registry_credentials.password", "ghp_secret-token"),
				),
			},
			// Third step: same config as step 2. Must be a no-op plan — proves the
			// Read-after-Update state matches the config, i.e. no perpetual diff on
			// any of the fields including registry_credentials.password.
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_service_instance" "test" {
  service_id                  = "` + serviceId + `"
  environment_id              = "` + envId + `"
  source_image                = "ghcr.io/owner/app@sha256:abc123"
  start_command               = "node server.js"
  healthcheck_path            = "/status"
  healthcheck_timeout         = 60
  restart_policy_type         = "ALWAYS"
  restart_policy_max_retries  = 5
  pre_deploy_command          = "seed"
  registry_credentials = {
    username = "myuser"
    password = "ghp_secret-token"
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
