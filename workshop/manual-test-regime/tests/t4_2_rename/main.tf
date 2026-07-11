# T4.2 — Rename lifecycle. Apply once to create; second apply renames
# service, environment, and inline volume in place. Each rename hits a
# distinct mutation surface (serviceUpdate, renameEnvironment, updateVolume).
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}

variable "svc_name" {
  type    = string
  default = "dev-svc"
}
variable "env_name" {
  type    = string
  default = "dev"
}
variable "vol_name" {
  type    = string
  default = "old-vol"
}

resource "railway_project" "acc" {
  name = "AAA-provctest-t4-2-rename"
  default_environment = { name = "core" }
}

resource "railway_environment" "env" {
  name                  = var.env_name
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_service" "svc" {
  name           = var.svc_name
  project_id     = railway_project.acc.id
  environment_id = railway_environment.env.id
  depends_on     = [railway_environment.env]
  volume = {
    name       = var.vol_name
    mount_path = "/data"
  }
}
