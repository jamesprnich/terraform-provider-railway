package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccServiceResourceConfigDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// Update and Read testing regions
			{
				Config: testAccServiceResourceConfigNonDefaultRegions("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "3"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "us-east4-eqdc4a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			// {
			// 	ResourceName:      "railway_service.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// Update and Read testing image
			{
				Config: testAccServiceResourceConfigNonDefaultImage("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", testAccDefaultRegion),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing repo
			// {
			// 	Config: testAccServiceResourceConfigNonDefaultRepo("nue-todo-app"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
			// 		resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo", "railwayapp/blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo_branch", "main"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "root_directory", "blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "config_path", "blog/railway.yaml"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
			// 	),
			// },
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing volume
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("nue-todo-app", "todo-app-volume", "/mnt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "todo-app-volume"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/mnt"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", testAccDefaultRegion),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", testAccDefaultRegion),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", testAccDefaultRegion),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", testAccDefaultRegion),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultRepo(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			// {
			// 	Config: testAccServiceResourceConfigNonDefaultRepo("todo-app"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
			// 		resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo", "railwayapp/blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo_branch", "main"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "root_directory", "blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "config_path", "blog/railway.yaml"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			//		resource.TestCheckResourceAttr("railway_service.test", "region", "us-west1"),
			//		resource.TestCheckResourceAttr("railway_service.test", "num_replicas", "1"),
			// 	),
			// },
			// // ImportState testing
			// {
			// 	ResourceName:      "railway_service.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// // Update with same values
			// {
			// 	Config: testAccServiceResourceConfigNonDefaultRepo("todo-app"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
			// 		resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo", "railwayapp/blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo_branch", "main"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "root_directory", "blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "config_path", "blog/railway.yaml"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			//		resource.TestCheckResourceAttr("railway_service.test", "region", "us-west1"),
			//		resource.TestCheckResourceAttr("railway_service.test", "num_replicas", "1"),
			// 	),
			// },
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultRegionsImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultRegionsImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "3"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "us-east4-eqdc4a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultRegionsImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "3"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "us-east4-eqdc4a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", testAccDefaultRegion),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultVolume(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", "todo-app-volume", "/mnt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "todo-app-volume"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/mnt"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", "todo-app-volume", "/mnt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "todo-app-volume"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/mnt"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with different values
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", "data-volume", "/data"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "data-volume"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/data"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", testAccProjectId),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceCronScheduleMultipleReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccServiceResourceConfigCronScheduleMultipleReplicas("todo-app"),
				ExpectError: regexp.MustCompile("(?s)`cron_schedule` can only be set when total number of replicas.*Found 2 replicas"),
			},
		},
	})
}

func TestAccServiceResourceCronScheduleMultipleRegions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccServiceResourceConfigCronScheduleMultipleRegions("todo-app"),
				ExpectError: regexp.MustCompile("(?s)`cron_schedule` can only be set when total number of replicas.*Found 2 replicas"),
			},
		},
	})
}

func TestAccServiceResource_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceResourceConfigDefault("disappears-test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					testAccCheckServiceDisappears("railway_service.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccServiceResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "%s"
}
`, name, testAccProjectId)
}

func testAccServiceResourceConfigNonDefaultRegions(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "%s"

  regions = [
    {
      region = "europe-west4-drams3a"
      num_replicas = 3
    },
    {
      region = "us-east4-eqdc4a"
      num_replicas = 2
    }
  ]
}
`, name, testAccProjectId)
}

func testAccServiceResourceConfigNonDefaultRegionsImage(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "%s"

  source_image = "hello-world"

  regions = [
    {
      region = "europe-west4-drams3a"
      num_replicas = 3
    },
    {
      region = "us-east4-eqdc4a"
      num_replicas = 2
    }
  ]
}
`, name, testAccProjectId)
}

func testAccServiceResourceConfigNonDefaultImage(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "%s"

  source_image = "hello-world"
}
`, name, testAccProjectId)
}

func testAccServiceResourceConfigNonDefaultRepo(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "%s"

  source_repo = "railwayapp/blog"
  source_repo_branch = "main"
  root_directory = "blog"
  config_path = "blog/railway.yaml"
}
`, name, testAccProjectId)
}

func testAccServiceResourceConfigNonDefaultVolume(name string, volumeName string, path string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "%s"

  cron_schedule = "0 0 * * *"

  volume = {
    name = "%s"
    mount_path = "%s"
  }
}
`, name, testAccProjectId, volumeName, path)
}

func testAccServiceResourceConfigCronScheduleMultipleReplicas(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name       = "%s"
  project_id = "%s"

  cron_schedule = "0 0 * * *"

  regions = [
    {
      region       = "europe-west4-drams3a"
      num_replicas = 2
    }
  ]
}
`, name, testAccProjectId)
}

func testAccServiceResourceConfigCronScheduleMultipleRegions(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name       = "%s"
  project_id = "%s"

  cron_schedule = "0 0 * * *"

  regions = [
    {
      region = "europe-west4-drams3a"
    },
    {
      region = "us-east4-eqdc4a"
    }
  ]
}
`, name, testAccProjectId)
}
