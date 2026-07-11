# E3 real collision: two services in the same project both asking for volume name "pgdata".
# Expected: first service succeeds, second service fails with "A volume named 'pgdata'
# already exists in this project". This is the failure mode that motivated the v0.11.1
# work — a genuine name collision on a project-scoped uniqueness constraint.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}
resource "railway_project" "acc" {
  name = "AAA-provctest-t4-3-collision"
  default_environment = { name = "core" }
}
resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}
resource "railway_service" "first" {
  name           = "dev-first"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  depends_on     = [railway_environment.dev]
  volume = {
    name       = "pgdata"
    mount_path = "/data-a"
  }
}
resource "railway_service" "second" {
  name           = "dev-second"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  depends_on     = [railway_environment.dev, railway_service.first]
  volume = {
    name       = "pgdata"
    mount_path = "/data-b"
  }
}
