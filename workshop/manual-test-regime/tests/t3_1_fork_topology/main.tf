# T3.1 — Fork topology mirroring TestAccLifecycle_forkTopology.
# Empty core + two forks (dev, prd). Services scoped to each fork.
# Standalone volume on the dev-side postgres, real Postgres image deploy.
# This is the same shape used by the acceptance test in
# lifecycle_acceptance_test.go, expressed as HCL so it runs from the CLI.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}

resource "railway_project" "acc" {
  name                = "AAA-provctest-t3-1-fork"
  default_environment = { name = "core" }
}

resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_environment" "prd" {
  name                  = "prd"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_service" "dev_postgres" {
  name           = "dev-postgres"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  depends_on     = [railway_environment.dev]
}

resource "railway_service_instance" "dev_postgres" {
  service_id     = railway_service.dev_postgres.id
  environment_id = railway_environment.dev.id
  source_image   = "postgres:17.5-alpine"
  vcpus          = 0.5
  memory_gb      = 0.25
}

resource "railway_volume" "dev_data" {
  project_id     = railway_project.acc.id
  service_id     = railway_service.dev_postgres.id
  environment_id = railway_environment.dev.id
  mount_path     = "/var/lib/postgresql/data"
  name           = "dev-postgres-data"
}

resource "railway_service" "prd_postgres" {
  name           = "prd-postgres"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.prd.id
  depends_on     = [railway_environment.prd]
}

resource "railway_service_instance" "prd_postgres" {
  service_id     = railway_service.prd_postgres.id
  environment_id = railway_environment.prd.id
  source_image   = "postgres:17.5-alpine"
  vcpus          = 0.5
  memory_gb      = 0.25
}
