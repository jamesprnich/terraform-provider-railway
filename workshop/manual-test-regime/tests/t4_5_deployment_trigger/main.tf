# T4.5 — railway_deployment_trigger against a public GitHub repo.
# Depends on the workspace having the Railway GitHub app connected to at
# least this repo (or a public repo access grant). If that connection is not
# set up, the create fails and the test is marked skipped rather than failed.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}

resource "railway_project" "acc" {
  name = "AAA-provctest-t4-5-trigger"
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

resource "railway_deployment_trigger" "gh" {
  project_id      = railway_project.acc.id
  environment_id  = railway_environment.dev.id
  service_id      = railway_service.svc.id
  repository      = "jamesprnich/terraform-provider-railway"
  branch          = "main"
  source_provider = "github"
}
