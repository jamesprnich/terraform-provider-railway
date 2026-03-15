package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ==================== Project Data Source ====================

func TestAccProjectDataSource_byId(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "railway_project" "test" {
  id = "%s"
}
`, testAccProjectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_project.test", "id", testAccProjectId),
					resource.TestCheckResourceAttr("data.railway_project.test", "name", testAccProjectName),
				),
			},
		},
	})
}

func TestAccProjectDataSource_byName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "railway_project" "test" {
  name = "%s"
}
`, testAccProjectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_project.test", "id", testAccProjectId),
					resource.TestCheckResourceAttr("data.railway_project.test", "name", testAccProjectName),
				),
			},
		},
	})
}

// ==================== Environment Data Source ====================

func TestAccEnvironmentDataSource_byId(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "railway_environment" "test" {
  id = "%s"
}
`, testAccEnvironmentId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_environment.test", "id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("data.railway_environment.test", "name", testAccEnvironmentName),
					resource.TestCheckResourceAttr("data.railway_environment.test", "project_id", testAccProjectId),
				),
			},
		},
	})
}

func TestAccEnvironmentDataSource_byName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "railway_environment" "test" {
  name       = "%s"
  project_id = "%s"
}
`, testAccEnvironmentName, testAccProjectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_environment.test", "id", testAccEnvironmentId),
					resource.TestCheckResourceAttr("data.railway_environment.test", "name", testAccEnvironmentName),
					resource.TestCheckResourceAttr("data.railway_environment.test", "project_id", testAccProjectId),
				),
			},
		},
	})
}

// ==================== Service Data Source ====================

func TestAccServiceDataSource_byId(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "railway_service" "test" {
  id = "%s"
}
`, testAccServiceId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_service.test", "id", testAccServiceId),
					resource.TestCheckResourceAttr("data.railway_service.test", "name", testAccServiceName),
					resource.TestCheckResourceAttr("data.railway_service.test", "project_id", testAccProjectId),
				),
			},
		},
	})
}

func TestAccServiceDataSource_byName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "railway_service" "test" {
  name       = "%s"
  project_id = "%s"
}
`, testAccServiceName, testAccProjectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_service.test", "id", testAccServiceId),
					resource.TestCheckResourceAttr("data.railway_service.test", "name", testAccServiceName),
					resource.TestCheckResourceAttr("data.railway_service.test", "project_id", testAccProjectId),
				),
			},
		},
	})
}
