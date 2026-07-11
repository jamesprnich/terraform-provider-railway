# =============================================================================
# Railway Reference Deployment — self-contained demo
# =============================================================================
# What this example does:
#
#   Creates a throwaway Railway project with the v0.11.0 fork-based scoping
#   pattern — an empty `core` environment plus a `dev` fork — then stands up
#   a Postgres + Flask application in `dev` to demonstrate the shape.
#
# What it is NOT:
#
#   A production template. It provisions billable Railway resources under
#   your account. When you are done experimenting, `tofu destroy` at the
#   bottom of this file removes everything, or delete the project from the
#   Railway dashboard.
#
# What you need:
#
#   1. A Railway account and an account token — see
#      https://docs.railway.com/reference/api and set the `RAILWAY_TOKEN`
#      environment variable.
#   2. OpenTofu >= 1.11 (or Terraform >= 1.0).
#   3. Nothing else. The example pulls a public Postgres image and points at
#      the public terraform-provider-railway repository for the Flask test
#      app — no GitHub token, no private image registry, no external
#      infrastructure needed.
#
# What it demonstrates:
#
#   • Project's default environment is named `core` and stays empty forever
#     — the "safety anchor" that makes fork-based scoping possible.
#   • `dev` is a fork of `core`. Every real environment in your topology
#     (dev/tst/qa/prd) is a fork of `core` by the same pattern.
#   • Every service has `environment_id` set to `dev` so it lives only in
#     `dev`. Under strict env-scoping (the default), omitting it fails at
#     plan time.
#   • Source, build, deploy, and resource-limit configuration lives on
#     `railway_service_instance` — Railway's per-environment configuration
#     surface. See docs/guides/multi-environment-scoping.md for the full
#     explainer.
#
# Usage:
#
#   export RAILWAY_TOKEN="your-account-token"
#   tofu init && tofu apply
#
#   When done experimenting:
#
#   tofu destroy
# =============================================================================

terraform {
  required_providers {
    railway = {
      source  = "jamesprnich/railway"
      version = "~> 0.11.0"
    }
  }
}

provider "railway" {
  # RAILWAY_TOKEN comes from your environment.
  # strict_env_scoping defaults to true.
}

# --- Variables ---
# Every default is set so `tofu apply` works with zero arguments. Override
# individually if you want a specific name or password.

variable "project_name" {
  type        = string
  description = "Throwaway project name. Prefix with AAA- so it sorts to the top of your Railway dashboard and is obviously a demo."
  default     = "AAA-provider-demo"
}

variable "postgres_password" {
  type        = string
  description = "Postgres password. Change this before running against any real environment."
  default     = "demo-only-not-a-real-password"
  sensitive   = true
}

# --- Project ---
# Default env is `core`. Kept empty forever. See the concept guide for why.

resource "railway_project" "demo" {
  name = var.project_name

  default_environment = {
    name = "core"
  }
}

locals {
  core_env_id = railway_project.demo.default_environment.id
}

# --- Fork environment: dev ---

resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.demo.id
  source_environment_id = local.core_env_id
}

# --- Services (empty shells scoped to dev) ---

resource "railway_service" "postgres" {
  name           = "dev-postgres"
  project_id     = railway_project.demo.id
  environment_id = railway_environment.dev.id

  # See docs/resources/service.md — `depends_on` is required because
  # railway_service references only project_id.
  depends_on = [railway_environment.dev]
}

# Persistent volume for postgres data.
#
# Prefer the standalone `railway_volume` over the inline `volume` block on
# `railway_service` when you want explicit lifecycle control — backup
# schedule, cross-service references, or independent destroy behaviour.
#
# Note the `name` — every resource in this file is env-prefixed (`dev-`)
# because Railway enforces name uniqueness *per project*, not per environment.
# When this pattern is grown to `dev`/`tst`/`prd`, each environment gets its
# own `<env>-postgres-data`, `<env>-postgres` service, `<env>-app` service,
# etc. — no name collisions, no ambiguity in the Railway dashboard.
resource "railway_volume" "postgres" {
  name           = "dev-postgres-data"
  project_id     = railway_project.demo.id
  service_id     = railway_service.postgres.id
  environment_id = railway_environment.dev.id
  mount_path     = "/var/lib/postgresql/data"
}

resource "railway_service" "app" {
  name           = "dev-app"
  project_id     = railway_project.demo.id
  environment_id = railway_environment.dev.id

  depends_on = [railway_environment.dev]
}

# --- Environment variables ---

resource "railway_variable_collection" "postgres" {
  environment_id = railway_environment.dev.id
  service_id     = railway_service.postgres.id

  variables = [
    { name = "PGDATA", value = "/var/lib/postgresql/data/pgdata" },
    { name = "POSTGRES_USER", value = "demo" },
    { name = "POSTGRES_PASSWORD", value = var.postgres_password },
    { name = "POSTGRES_DB", value = "demo" },
    { name = "PORT", value = "5432" },
  ]
}

resource "railway_variable_collection" "app" {
  environment_id = railway_environment.dev.id
  service_id     = railway_service.app.id

  variables = [
    { name = "PORT", value = "8080" },
    { name = "DATABASE_URL", value = "postgresql://demo:${var.postgres_password}@dev-postgres.railway.internal:5432/demo?connect_timeout=5" },
  ]
}

# --- Service instances (per-env source and config) ---
# This is where source, build, and deploy configuration lives. Railway's
# serviceInstanceUpdate is env-scoped, so nothing here can leak out of dev.

resource "railway_service_instance" "postgres" {
  service_id     = railway_service.postgres.id
  environment_id = railway_environment.dev.id
  source_image   = "postgres:17.5-alpine"
  vcpus          = 0.5
  memory_gb      = 0.25
}

resource "railway_service_instance" "app" {
  service_id     = railway_service.app.id
  environment_id = railway_environment.dev.id

  # Points at THIS repository's test-app example (a small Flask app). The
  # repo is public so no GitHub token is required. Railway will build from
  # the repo's default branch. If you fork the repo, change source_repo to
  # your fork.
  source_repo    = "jamesprnich/terraform-provider-railway"
  root_directory = "examples/test-app"

  vcpus            = 0.5
  memory_gb        = 0.25
  healthcheck_path = "/health"
}

# --- Public domain for the app ---

resource "railway_service_domain" "app" {
  service_id     = railway_service.app.id
  environment_id = railway_environment.dev.id

  depends_on = [railway_service_instance.app]
}

# --- Outputs ---

output "project_id" {
  value = railway_project.demo.id
}

output "core_environment_id" {
  description = "Empty core env — anchor for the fork pattern. Export this if you extend the example to multiple envs (dev/tst/qa/prd)."
  value       = local.core_env_id
}

output "dev_environment_id" {
  value = railway_environment.dev.id
}

output "app_url" {
  description = "Public URL for the deployed Flask test app. May take a couple of minutes to become live after apply."
  value       = "https://${railway_service_domain.app.domain}"
}
