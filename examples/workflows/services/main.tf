# =============================================================================
# Example: Services Layer
# =============================================================================
# Creates the Railway project and empty services. No source images, no source
# repos — the environment layer controls what runs in each environment.
#
# See docs/guides/two-layer-architecture.md for why services are empty.
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

# --- Project ---

resource "railway_project" "main" {
  name = var.project_name
}

# --- Services (empty shells — configured per-environment in the env layer) ---

resource "railway_service" "app" {
  name       = "app"
  project_id = railway_project.main.id
}

resource "railway_service" "postgres" {
  name       = "postgres"
  project_id = railway_project.main.id
}

# --- Outputs ---

output "project_id" {
  value = railway_project.main.id
}

output "project_name" {
  value = railway_project.main.name
}

# --- Variables ---

variable "project_name" {
  type        = string
  description = "Name of the Railway project."
  default     = "test-app"
}
