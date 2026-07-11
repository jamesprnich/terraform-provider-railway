# E4 flake test — parametrized by suffix so back-to-back runs use distinct
# project names. Same shape as E1 baseline; goal is 5 clean runs in a row.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}
variable "suffix" { type = string }
resource "railway_project" "acc" {
  name = "AAA-provctest-t4-1-flake-${var.suffix}"
  default_environment = { name = "core" }
}
resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}
resource "railway_service" "postgres" {
  name           = "dev-postgres"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  depends_on     = [railway_environment.dev]
  volume = {
    name       = "pgdata"
    mount_path = "/var/lib/postgresql/data"
  }
}
