# T1.2 — railway_environment fork under strict, non-fork under permissive.
# Uses provider blocks with aliases so both scoping modes are exercised
# against distinct projects in one apply.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}
provider "railway" {
  alias              = "permissive"
  strict_env_scoping = false
}

resource "railway_project" "strict_proj" {
  name = "AAA-provctest-t1-2-strict"
  default_environment = { name = "core" }
}

# Fork under strict mode.
resource "railway_environment" "dev_fork" {
  name                  = "dev"
  project_id            = railway_project.strict_proj.id
  source_environment_id = railway_project.strict_proj.default_environment.id
}

resource "railway_project" "permissive_proj" {
  provider = railway.permissive
  name     = "AAA-provctest-t1-2-permissive"
  default_environment = { name = "core" }
}

# Non-fork under permissive mode.
resource "railway_environment" "nonfork" {
  provider   = railway.permissive
  name       = "loose"
  project_id = railway_project.permissive_proj.id
}
