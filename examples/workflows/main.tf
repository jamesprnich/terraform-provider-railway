# =============================================================================
# Railway Test App — Full Stack
# =============================================================================
# Deploys a Flask app + Postgres on Railway with private networking.
# Proves: project, services, variables, volume, service instances, domain.
#
# Usage:
#   tofu apply -var='app_repo=owner/repo' -var='postgres_password=xxx'
# =============================================================================

terraform {
  required_providers {
    railway = {
      source  = "terraform-community-providers/railway"
      version = "~> 0.7.0"
    }
  }
}

provider "railway" {
  # Set via RAILWAY_TOKEN environment variable
}

# --- Variables ---

variable "project_name" {
  type        = string
  description = "Name of the Railway project."
  default     = "test-app"
}

variable "app_repo" {
  type        = string
  description = "GitHub repo for the Flask test app (e.g. owner/railway-terraform-provider)."
}

variable "postgres_password" {
  type        = string
  sensitive   = true
  description = "Password for the Postgres database."
}

# --- Project ---

resource "railway_project" "main" {
  name = var.project_name

  default_environment = {
    name = "dev"
  }
}

locals {
  environment_id = railway_project.main.default_environment.id
}

# --- Services ---

resource "railway_service" "postgres" {
  name         = "postgres-dev"
  project_id   = railway_project.main.id
  source_image = "postgres:17.5-alpine"

  volume = {
    name       = "pgdata"
    mount_path = "/data"
  }
}

resource "railway_service" "app" {
  name               = "app-dev"
  project_id         = railway_project.main.id
  source_repo        = var.app_repo
  source_repo_branch = "main"
  root_directory     = "examples/test-app"
}

# --- Postgres variables ---

resource "railway_variable" "pgdata" {
  name           = "PGDATA"
  value          = "/data/pgdata"
  environment_id = local.environment_id
  service_id     = railway_service.postgres.id
}

resource "railway_variable" "postgres_user" {
  name           = "POSTGRES_USER"
  value          = "testapp"
  environment_id = local.environment_id
  service_id     = railway_service.postgres.id
}

resource "railway_variable" "postgres_password" {
  name           = "POSTGRES_PASSWORD"
  value          = var.postgres_password
  environment_id = local.environment_id
  service_id     = railway_service.postgres.id
}

resource "railway_variable" "postgres_db" {
  name           = "POSTGRES_DB"
  value          = "testapp"
  environment_id = local.environment_id
  service_id     = railway_service.postgres.id
}

resource "railway_variable" "postgres_port" {
  name           = "PORT"
  value          = "5432"
  environment_id = local.environment_id
  service_id     = railway_service.postgres.id
}

# --- App variables ---

resource "railway_variable" "app_port" {
  name           = "PORT"
  value          = "8080"
  environment_id = local.environment_id
  service_id     = railway_service.app.id
}

resource "railway_variable" "database_url" {
  name           = "DATABASE_URL"
  value          = "postgresql://testapp:${var.postgres_password}@postgres-dev.railway.internal:5432/testapp"
  environment_id = local.environment_id
  service_id     = railway_service.app.id
}

# --- Service instance config ---

resource "railway_service_instance" "postgres" {
  service_id     = railway_service.postgres.id
  environment_id = local.environment_id
  vcpus          = 0.5
  memory_gb      = 0.25
}

resource "railway_service_instance" "app" {
  service_id       = railway_service.app.id
  environment_id   = local.environment_id
  vcpus            = 0.5
  memory_gb        = 0.25
  healthcheck_path = "/health"
}

# --- Public domain ---

resource "railway_service_domain" "app" {
  service_id     = railway_service.app.id
  environment_id = local.environment_id

  depends_on = [railway_service_instance.app]
}

# --- Outputs ---

output "project_id" {
  value = railway_project.main.id
}

output "environment_id" {
  value = local.environment_id
}

output "app_url" {
  value = "https://${railway_service_domain.app.domain}"
}
