# T1.3 — Data sources: lookup project + env + service by id AND by name.
# First apply creates the fixtures; second apply adds the data sources
# that reference them. Kept in one apply here since data sources are
# resolved after resources on the same graph.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}

resource "railway_project" "acc" {
  name = "AAA-provctest-t1-3"
  default_environment = { name = "core" }
}

resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_service" "svc" {
  name           = "dev-svc"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  depends_on     = [railway_environment.dev]
}

# Data source lookups by id.
data "railway_project" "by_id" { id = railway_project.acc.id }
data "railway_environment" "by_id" { id = railway_environment.dev.id }
data "railway_service" "by_id" { id = railway_service.svc.id }

# Data source lookups by name.
data "railway_project" "by_name" { name = railway_project.acc.name }
data "railway_environment" "by_name" {
  project_id = railway_project.acc.id
  name       = railway_environment.dev.name
}
data "railway_service" "by_name" {
  project_id = railway_project.acc.id
  name       = railway_service.svc.name
}

output "proj_by_id_name"    { value = data.railway_project.by_id.name }
output "proj_by_name_id"    { value = data.railway_project.by_name.id }
output "env_by_id_name"     { value = data.railway_environment.by_id.name }
output "env_by_name_id"     { value = data.railway_environment.by_name.id }
output "svc_by_id_name"     { value = data.railway_service.by_id.name }
output "svc_by_name_id"     { value = data.railway_service.by_name.id }
