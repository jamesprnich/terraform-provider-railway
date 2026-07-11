# T4.4 — Rapid apply-destroy back-to-back on a config that triggers
# variable_collection redeploy on Delete. Stresses the redeploy fix under
# tight timing: build is still in-flight from the create when destroy runs.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}
variable "suffix" { type = string }
resource "railway_project" "acc" {
  name = "AAA-provctest-t4-4-${var.suffix}"
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
resource "railway_service_instance" "svc" {
  service_id     = railway_service.svc.id
  environment_id = railway_environment.dev.id
  source_image   = "nginx:alpine"
  vcpus          = 0.5
  memory_gb      = 0.25
}
resource "railway_variable_collection" "vars" {
  environment_id = railway_environment.dev.id
  service_id     = railway_service.svc.id
  variables = [
    { name = "FOO", value = "bar" },
    { name = "BAZ", value = "qux" },
  ]
}
