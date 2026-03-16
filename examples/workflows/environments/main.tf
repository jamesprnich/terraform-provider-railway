# =============================================================================
# Config Layer
# =============================================================================
# Configures environment variables, domains, volumes, and service instance
# settings. Safe to destroy and re-apply — no source changes.
# NOTE: Destroying this layer deletes volumes (and their data).
#
# Requires the infrastructure layer to be applied first.
# Pass the environment_id from infrastructure outputs.
#
# The test app is a Flask "Hello World" that reads a message from Postgres,
# proving full-stack connectivity via Railway private networking.
# Source: examples/test-app/
#
# Usage:
#   tofu apply -var='environment_id=xxx' -var='postgres_password=xxx'
# =============================================================================

terraform {
  required_providers {
    railway = {
      source  = "jamesprnich/railway"
      version = "~> 0.8.0"
    }
  }
}

provider "railway" {
  # Set via RAILWAY_TOKEN environment variable (account token required)
}

# --- Discover infrastructure ---

data "railway_project" "main" {
  name = var.project_name
}

data "railway_service" "app" {
  project_id = data.railway_project.main.id
  name       = "app-dev"
}

data "railway_service" "postgres" {
  project_id = data.railway_project.main.id
  name       = "postgres-dev"
}

# --- Postgres variables ---

# PGDATA must point to a subdirectory of the mount to avoid lost+found conflict
resource "railway_variable" "pgdata" {
  name           = "PGDATA"
  value          = "/data/pgdata"
  environment_id = var.environment_id
  service_id     = data.railway_service.postgres.id
}

resource "railway_variable" "postgres_user" {
  name           = "POSTGRES_USER"
  value          = "testapp"
  environment_id = var.environment_id
  service_id     = data.railway_service.postgres.id
}

resource "railway_variable" "postgres_password" {
  name           = "POSTGRES_PASSWORD"
  value          = var.postgres_password
  environment_id = var.environment_id
  service_id     = data.railway_service.postgres.id
}

resource "railway_variable" "postgres_db" {
  name           = "POSTGRES_DB"
  value          = "testapp"
  environment_id = var.environment_id
  service_id     = data.railway_service.postgres.id
}

# Railway private networking requires PORT for Docker image services
resource "railway_variable" "postgres_port" {
  name           = "PORT"
  value          = "5432"
  environment_id = var.environment_id
  service_id     = data.railway_service.postgres.id
}

# --- Postgres volume ---
# Volume is in the config layer (not infrastructure) so it's created alongside
# the Postgres variables. This ensures initdb runs with the correct credentials
# when the data directory is first populated.

resource "railway_volume" "pgdata" {
  name           = "pgdata"
  project_id     = data.railway_project.main.id
  service_id     = data.railway_service.postgres.id
  environment_id = var.environment_id
  mount_path     = "/data"

  depends_on = [
    railway_variable.pgdata,
    railway_variable.postgres_user,
    railway_variable.postgres_password,
    railway_variable.postgres_db,
    railway_variable.postgres_port,
  ]
}

# --- App variables ---

resource "railway_variable" "app_port" {
  name           = "PORT"
  value          = "8080"
  environment_id = var.environment_id
  service_id     = data.railway_service.app.id
}

resource "railway_variable" "database_url" {
  name           = "DATABASE_URL"
  value          = "postgresql://testapp:${var.postgres_password}@postgres-dev.railway.internal:5432/testapp"
  environment_id = var.environment_id
  service_id     = data.railway_service.app.id
}

# --- Service instance config (limits, healthchecks) ---

resource "railway_service_instance" "postgres" {
  service_id     = data.railway_service.postgres.id
  environment_id = var.environment_id
  vcpus          = 0.5
  memory_gb      = 0.25
}

resource "railway_service_instance" "app" {
  service_id       = data.railway_service.app.id
  environment_id   = var.environment_id
  vcpus            = 0.5
  memory_gb        = 0.25
  healthcheck_path = "/health"
}

# --- Public domain for the app ---

resource "railway_service_domain" "app" {
  service_id     = data.railway_service.app.id
  environment_id = var.environment_id

  depends_on = [railway_service_instance.app]
}

# --- Outputs ---

output "app_url" {
  value = "https://${railway_service_domain.app.domain}"
}

# --- Variables ---

variable "project_name" {
  type        = string
  description = "Name of the Railway project (must match infrastructure layer)."
  default     = "test-app"
}

variable "environment_id" {
  type        = string
  description = "Railway environment ID from infrastructure layer outputs."
}

variable "postgres_password" {
  type        = string
  sensitive   = true
  description = "Password for the Postgres database."
}
