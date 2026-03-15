# =============================================================================
# Infrastructure Layer
# =============================================================================
# Creates the Railway project, environments, services with sources, and volumes.
# Each environment gets its own services (separate-services-per-environment).
#
# Source connections (serviceConnect) are service-level, so per-env services
# prevent cross-environment contamination.
#
# Rarely modified. Destroying this destroys all infrastructure and data.
#
# Usage:
#   tofu apply -var='app_repo=owner/repo'
# =============================================================================

terraform {
  required_providers {
    railway = {
      source  = "terraform-community-providers/railway"
      version = "~> 0.8.0"
    }
  }
}

provider "railway" {
  # Set via RAILWAY_TOKEN environment variable (account token required)
}

# --- Project ---
# Default environment is named "dev" instead of "production" so serviceConnect
# only deploys there. One project, one environment, per-env services.

resource "railway_project" "main" {
  name = var.project_name

  default_environment = {
    name = "dev"
  }
}

# --- Postgres (dev) ---
# Docker image source, with a persistent volume for data.

resource "railway_service" "postgres_dev" {
  name         = "postgres-dev"
  project_id   = railway_project.main.id
  source_image = "postgres:17.5-alpine"
}

# --- App (dev) ---
# GitHub repo source, points to the test-app Flask app.

resource "railway_service" "app_dev" {
  name               = "app-dev"
  project_id         = railway_project.main.id
  source_repo        = var.app_repo
  source_repo_branch = "main"
  root_directory     = "examples/test-app"
}

# --- Outputs ---

output "project_id" {
  value = railway_project.main.id
}

output "project_name" {
  value = railway_project.main.name
}

output "dev_environment_id" {
  value = railway_project.main.default_environment.id
}

output "postgres_dev_service_id" {
  value = railway_service.postgres_dev.id
}

output "app_dev_service_id" {
  value = railway_service.app_dev.id
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
