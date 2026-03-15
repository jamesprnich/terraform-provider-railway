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

func TestAccVolumeResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVolumeResourceConfig("/data/acc-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_volume.test", "id"),
					resource.TestCheckResourceAttr("railway_volume.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_volume.test", "service_id", testAccServiceId),
					resource.TestCheckResourceAttr("railway_volume.test", "environment_id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("railway_volume.test", "mount_path", "/data/acc-test"),
					resource.TestCheckResourceAttrSet("railway_volume.test", "size_mb"),
				),
			},
			// Import
			{
				ResourceName: "railway_volume.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["railway_volume.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return testAccProjectId + ":" + rs.Primary.ID + ":" + testAccServiceId + ":" + testAccEnvironmentId, nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccVolumeResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVolumeResourceConfig("/data/acc-disappears"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_volume.test", "id"),
					testAccCheckVolumeDisappears("railway_volume.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccVolumeResourceConfig(mountPath string) string {
	return fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "%s"
}
`, testAccProjectId, testAccServiceId, testAccEnvironmentId, mountPath)
}

func TestVolumeResource_basic(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	volumeCreateResp := fmt.Sprintf(`{"data":{"volumeCreate":{"id":"vol-123","name":"my-volume","projectId":"%s"}}}`, projectId)
	volumeInstancesResp := fmt.Sprintf(`{"data":{"project":{"volumes":{"edges":[{"node":{"id":"vol-123","name":"my-volume","volumeInstances":{"edges":[{"node":{"id":"vi-123","environmentId":"%s","serviceId":"%s","mountPath":"/data","sizeMB":1024}}]}}}]}}}}`, envId, serviceId)

	srv := newMockGraphQLServer(t, mockFixtures{
		"createVolume":       volumeCreateResp,
		"getVolumeInstances": volumeInstancesResp,
		"deleteVolume":       `{"data":{"volumeDelete":true}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/data"
}
`, projectId, serviceId, envId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "id", "vol-123"),
					resource.TestCheckResourceAttr("railway_volume.test", "volume_instance_id", "vi-123"),
					resource.TestCheckResourceAttr("railway_volume.test", "name", "my-volume"),
					resource.TestCheckResourceAttr("railway_volume.test", "project_id", projectId),
					resource.TestCheckResourceAttr("railway_volume.test", "service_id", serviceId),
					resource.TestCheckResourceAttr("railway_volume.test", "environment_id", envId),
					resource.TestCheckResourceAttr("railway_volume.test", "mount_path", "/data"),
					resource.TestCheckResourceAttr("railway_volume.test", "size_mb", "1024"),
				),
			},
		},
	})
}

func TestVolumeResource_withName(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	// Server returns default name "Volume", then we update to custom name
	volumeInstancesResp := fmt.Sprintf(`{"data":{"project":{"volumes":{"edges":[{"node":{"id":"vol-456","name":"postgres-data","volumeInstances":{"edges":[{"node":{"id":"vi-456","environmentId":"%s","serviceId":"%s","mountPath":"/var/lib/postgresql/data","sizeMB":2048}}]}}}]}}}}`, envId, serviceId)

	var mu sync.Mutex
	created := false

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
		case "createVolume":
			created = true
			fmt.Fprintf(w, `{"data":{"volumeCreate":{"id":"vol-456","name":"Volume","projectId":"%s"}}}`, projectId)
		case "updateVolume":
			fmt.Fprint(w, `{"data":{"volumeUpdate":{"id":"vol-456","name":"postgres-data"}}}`)
		case "getVolumeInstances":
			if created {
				fmt.Fprint(w, volumeInstancesResp)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"errors":[{"message":"not found"}]}`)
			}
		case "deleteVolume":
			fmt.Fprint(w, `{"data":{"volumeDelete":true}}`)
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
				Config: testUnitProviderConfig(server.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/var/lib/postgresql/data"
  name           = "postgres-data"
}
`, projectId, serviceId, envId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "id", "vol-456"),
					resource.TestCheckResourceAttr("railway_volume.test", "name", "postgres-data"),
					resource.TestCheckResourceAttr("railway_volume.test", "mount_path", "/var/lib/postgresql/data"),
					resource.TestCheckResourceAttr("railway_volume.test", "size_mb", "2048"),
				),
			},
		},
	})
}

func TestVolumeResource_disappears(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	volumeCreateResp := fmt.Sprintf(`{"data":{"volumeCreate":{"id":"vol-dis","name":"my-volume","projectId":"%s"}}}`, projectId)
	volumeInstancesResp := fmt.Sprintf(`{"data":{"project":{"volumes":{"edges":[{"node":{"id":"vol-dis","name":"my-volume","volumeInstances":{"edges":[{"node":{"id":"vi-123","environmentId":"%s","serviceId":"%s","mountPath":"/data","sizeMB":1024}}]}}}]}}}}`, envId, serviceId)

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createVolume":       volumeCreateResp,
		"getVolumeInstances": volumeInstancesResp,
		"deleteVolume":       `{"data":{"volumeDelete":true}}`,
	}, "getVolumeInstances")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/data"
}
`, projectId, serviceId, envId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "id", "vol-dis"),
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

func TestVolumeResource_import(t *testing.T) {
	projectId := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	serviceId := "11111111-2222-3333-4444-555555555555"
	envId := "22222222-3333-4444-5555-666666666666"

	volumeInstancesResp := fmt.Sprintf(`{"data":{"project":{"volumes":{"edges":[{"node":{"id":"vol-789","name":"my-volume","volumeInstances":{"edges":[{"node":{"id":"vi-789","environmentId":"%s","serviceId":"%s","mountPath":"/data","sizeMB":512}}]}}}]}}}}`, envId, serviceId)

	srv := newMockGraphQLServer(t, mockFixtures{
		"createVolume":       fmt.Sprintf(`{"data":{"volumeCreate":{"id":"vol-789","name":"my-volume","projectId":"%s"}}}`, projectId),
		"getVolumeInstances": volumeInstancesResp,
		"deleteVolume":       `{"data":{"volumeDelete":true}}`,
	})
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + fmt.Sprintf(`
resource "railway_volume" "test" {
  project_id     = "%s"
  service_id     = "%s"
  environment_id = "%s"
  mount_path     = "/data"
}
`, projectId, serviceId, envId),
			},
			{
				ResourceName:      "railway_volume.test",
				ImportState:       true,
				ImportStateId:     projectId + ":vol-789:" + serviceId + ":" + envId,
				ImportStateVerify: true,
			},
		},
	})
}
