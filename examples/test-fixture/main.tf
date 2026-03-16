terraform {
  required_providers {
    railway = {
      source = "jamesprnich/railway"
    }
  }
}

provider "railway" {}

# Persistent fixture project for acceptance tests.
# Other tests (environment, service, variable, etc.) reference this project.
resource "railway_project" "fixture" {
  name = "acc-test-fixture"
}

resource "railway_service" "fixture" {
  name       = "acc-test-service"
  project_id = railway_project.fixture.id
}

output "project_id" {
  value = railway_project.fixture.id
}

output "environment_id" {
  value = railway_project.fixture.default_environment.id
}

output "service_id" {
  value = railway_service.fixture.id
}
