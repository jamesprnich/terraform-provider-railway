# T1.4 — Plan-only test. Strict mode + missing environment_id on service
# and missing source_environment_id on environment MUST fail at plan time.
# We DO NOT apply — this is validated by `tofu plan` returning non-zero.
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}  # strict by default

resource "railway_project" "acc" {
  name = "AAA-provctest-t1-4"
  default_environment = { name = "core" }
}

# Missing environment_id — strict mode should reject.
resource "railway_service" "bad_svc" {
  name       = "bad-svc"
  project_id = railway_project.acc.id
}

# Missing source_environment_id — strict mode should reject.
resource "railway_environment" "bad_env" {
  name       = "bad-env"
  project_id = railway_project.acc.id
}
