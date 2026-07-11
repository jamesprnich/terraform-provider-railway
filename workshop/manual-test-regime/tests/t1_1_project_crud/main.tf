# T1.1 — railway_project CRUD, rename in-place.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}
variable "project_name" {
  type    = string
  default = "AAA-provctest-t1-1"
}
resource "railway_project" "acc" {
  name        = var.project_name
  description = "provider-comprehensive-test tier 1.1"
  default_environment = { name = "core" }
}
output "project_id" { value = railway_project.acc.id }
output "default_env_id" { value = railway_project.acc.default_environment.id }
