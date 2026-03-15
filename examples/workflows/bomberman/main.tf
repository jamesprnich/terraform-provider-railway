# =============================================================================
# Atomic Bomberman — Railway Deployment
# =============================================================================
# Deploys a Colyseus game server + React client on Railway.
#
# Architecture:
#   - Colyseus server: Node.js WebSocket game server (server/ directory)
#   - React client: Vite-built SPA served via `serve` (client/ directory)
#   - Both deploy from the same GitHub repo with different root directories
#
# Usage:
#   export RAILWAY_TOKEN="<token>"
#   tofu apply
#
# Verify:
#   curl $(tofu output -raw server_url)/health
#   Open $(tofu output -raw client_url) in browser
#
# Tear down:
#   tofu destroy
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
  # Set via RAILWAY_TOKEN environment variable
}

# --- Variables ---

variable "project_name" {
  type        = string
  description = "Name of the Railway project."
  default     = "bomberman"
}

variable "repo" {
  type        = string
  description = "GitHub repo (owner/name)."
  default     = "jamesprnich/atomic-bomberman"
}

variable "branch" {
  type        = string
  description = "Branch to deploy from."
  default     = "master"
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

resource "railway_service" "server" {
  name               = "colyseus-server"
  project_id         = railway_project.main.id
  source_repo        = var.repo
  source_repo_branch = var.branch
  root_directory     = "server"
  config_path        = "server/railway.toml"
}

resource "railway_service" "client" {
  name               = "web-client"
  project_id         = railway_project.main.id
  source_repo        = var.repo
  source_repo_branch = var.branch
  root_directory     = "client"
  config_path        = "client/railway.toml"
}

# --- Public Domains ---
# Server domain must be created before client variables (client needs the URL)
# Note: deployment triggers are NOT needed here because railway_service with
# source_repo automatically creates a trigger. Creating a second would fail.

resource "railway_service_domain" "server" {
  service_id     = railway_service.server.id
  environment_id = local.environment_id
}

resource "railway_service_domain" "client" {
  service_id     = railway_service.client.id
  environment_id = local.environment_id
}

# --- Server Variables ---

resource "railway_variable" "server_port" {
  name           = "PORT"
  value          = "2567"
  environment_id = local.environment_id
  service_id     = railway_service.server.id
}

resource "railway_variable" "server_app_env" {
  name           = "APP_ENV"
  value          = "DEV"
  environment_id = local.environment_id
  service_id     = railway_service.server.id
}

resource "railway_variable" "server_cors_origin" {
  name           = "CORS_ORIGIN"
  value          = "https://${railway_service_domain.client.domain}"
  environment_id = local.environment_id
  service_id     = railway_service.server.id
}

# --- Client Variables ---
# Railway injects env vars during Docker build. The client Dockerfile
# declares ARG VITE_COLYSEUS_URL which Vite bakes into the static
# bundle at build time.

resource "railway_variable" "client_port" {
  name           = "PORT"
  value          = "3000"
  environment_id = local.environment_id
  service_id     = railway_service.client.id
}

resource "railway_variable" "client_colyseus_url" {
  name           = "VITE_COLYSEUS_URL"
  value          = "wss://${railway_service_domain.server.domain}"
  environment_id = local.environment_id
  service_id     = railway_service.client.id
}

resource "railway_variable" "client_app_env" {
  name           = "VITE_APP_ENV"
  value          = "DEV"
  environment_id = local.environment_id
  service_id     = railway_service.client.id
}

# --- Service Instance Config ---

resource "railway_service_instance" "server" {
  service_id        = railway_service.server.id
  environment_id    = local.environment_id
  region            = "asia-southeast1"
  vcpus             = 0.5
  memory_gb         = 0.25
  sleep_application = true
  overlap_seconds   = 2
  draining_seconds  = 3
}

resource "railway_service_instance" "client" {
  service_id        = railway_service.client.id
  environment_id    = local.environment_id
  region            = "asia-southeast1"
  vcpus             = 0.5
  memory_gb         = 0.25
  sleep_application = true
  overlap_seconds   = 2
  draining_seconds  = 3
}

# --- Outputs ---

output "project_id" {
  value = railway_project.main.id
}

output "environment_id" {
  value = local.environment_id
}

output "server_url" {
  value = "https://${railway_service_domain.server.domain}"
}

output "client_url" {
  value = "https://${railway_service_domain.client.domain}"
}
