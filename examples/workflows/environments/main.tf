# =============================================================================
# Example: Environment Layer
# =============================================================================
# Configures a Railway environment with service instances, a Postgres volume,
# environment variables, and a public domain for the app.
#
# Uses data sources to discover services by name — no shared tfvars needed.
# Uses OpenTofu workspaces for per-environment isolation (dev, qa, prd).
#
# The test app is a Flask "Hello World" that reads a message from Postgres,
# proving full-stack connectivity via Railway private networking.
# Source: examples/test-app/
#
# Usage:
#   tofu workspace select -or-create dev
#   tofu plan
#   tofu apply
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
  # Set via RAILWAY_TOKEN environment variable (account token required)
}

# --- Per-environment config ---

locals {
  env = terraform.workspace

  config = {
    dev = {
      branch         = "main"
      subdomain      = "test-app-dev"
      vcpus          = 0.5
      memory_gb      = 0.25
      postgres_image = "postgres:17.5-alpine"
    }
    qa = {
      branch         = "main"
      subdomain      = "test-app-qa"
      vcpus          = 0.5
      memory_gb      = 0.25
      postgres_image = "postgres:17.5-alpine"
    }
    prd = {
      branch         = "main"
      subdomain      = "test-app"
      vcpus          = 0.5
      memory_gb      = 0.5
      postgres_image = "postgres:17.5-alpine"
    }
  }

  env_config = local.config[local.env]
}

# --- Discover services via data sources ---

data "railway_project" "main" {
  name = var.project_name
}

data "railway_service" "app" {
  project_id = data.railway_project.main.id
  name       = "app"
}

data "railway_service" "postgres" {
  project_id = data.railway_project.main.id
  name       = "postgres"
}

# --- Environment ---

resource "railway_environment" "this" {
  name       = local.env
  project_id = data.railway_project.main.id
}

# --- Postgres ---

resource "railway_service_instance" "postgres" {
  service_id     = data.railway_service.postgres.id
  environment_id = railway_environment.this.id
  source_image   = local.env_config.postgres_image
  vcpus          = local.env_config.vcpus
  memory_gb      = local.env_config.memory_gb
}

resource "railway_volume" "pgdata" {
  name           = "pgdata"
  project_id     = data.railway_project.main.id
  service_id     = data.railway_service.postgres.id
  environment_id = railway_environment.this.id
  mount_path     = "/data"

  depends_on = [railway_service_instance.postgres]
}

# PGDATA must point to a subdirectory of the mount to avoid lost+found conflict
resource "railway_variable" "pgdata" {
  name           = "PGDATA"
  value          = "/data/pgdata"
  environment_id = railway_environment.this.id
  service_id     = data.railway_service.postgres.id
}

resource "railway_variable" "postgres_user" {
  name           = "POSTGRES_USER"
  value          = "testapp"
  environment_id = railway_environment.this.id
  service_id     = data.railway_service.postgres.id
}

resource "railway_variable" "postgres_password" {
  name           = "POSTGRES_PASSWORD"
  value          = var.postgres_password
  environment_id = railway_environment.this.id
  service_id     = data.railway_service.postgres.id
}

resource "railway_variable" "postgres_db" {
  name           = "POSTGRES_DB"
  value          = "testapp"
  environment_id = railway_environment.this.id
  service_id     = data.railway_service.postgres.id
}

# Railway private networking requires PORT for Docker image services
resource "railway_variable" "postgres_port" {
  name           = "PORT"
  value          = "5432"
  environment_id = railway_environment.this.id
  service_id     = data.railway_service.postgres.id
}

# --- App (Flask test app from examples/test-app/) ---

resource "railway_service_instance" "app" {
  service_id     = data.railway_service.app.id
  environment_id = railway_environment.this.id
  source_repo    = var.app_repo
  root_directory = "examples/test-app"
  vcpus          = local.env_config.vcpus
  memory_gb      = local.env_config.memory_gb
  healthcheck_path = "/health"
}

resource "railway_variable" "database_url" {
  name           = "DATABASE_URL"
  value          = "postgresql://testapp:${var.postgres_password}@postgres.railway.internal:5432/testapp"
  environment_id = railway_environment.this.id
  service_id     = data.railway_service.app.id
}

# Commented out until Railway GitHub integration is configured for this workspace
# resource "railway_deployment_trigger" "app" {
#   service_id      = data.railway_service.app.id
#   environment_id  = railway_environment.this.id
#   project_id      = data.railway_project.main.id
#   repository      = var.app_repo
#   branch          = local.env_config.branch
#   root_directory  = "examples/test-app"
#   source_provider = "github"
#
#   depends_on = [railway_service_instance.app]
# }

# --- Public domain for the app ---

resource "railway_service_domain" "app" {
  subdomain      = local.env_config.subdomain
  service_id     = data.railway_service.app.id
  environment_id = railway_environment.this.id

  depends_on = [railway_service_instance.app]
}

# --- Outputs ---

output "environment_id" {
  value = railway_environment.this.id
}

output "app_url" {
  value = "https://${railway_service_domain.app.domain}"
}

# --- Variables ---

variable "project_name" {
  type        = string
  description = "Name of the Railway project (must match services layer)."
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
