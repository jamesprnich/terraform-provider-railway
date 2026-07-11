# Tier 2 — exercise every non-workspace resource in one apply.
# NO railway_service_instance = no billable compute.
# Skipped: project_member, ssh_public_key, trusted_domain (workspace-mutating);
#          custom_domain (DNS); bucket (Delete is no-op).
terraform {
  required_providers {
    railway = { source = "jamesprnich/railway" }
  }
}
provider "railway" {}

variable "workspace_id" { type = string }

resource "railway_project" "acc" {
  name                = "AAA-provctest-t2"
  description         = "provider comprehensive test tier 2"
  default_environment = { name = "core" }
}

resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.acc.id
  source_environment_id = railway_project.acc.default_environment.id
}

resource "railway_service" "svc" {
  name           = "dev-svc"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  icon           = "🐹"
  depends_on     = [railway_environment.dev]
}

# ---- Variables ----
resource "railway_shared_variable" "sentry" {
  name           = "SENTRY_DSN"
  value          = "https://key@sentry.io/123"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
}

resource "railway_variable" "single" {
  name           = "APP_MODE"
  value          = "test"
  environment_id = railway_environment.dev.id
  service_id     = railway_service.svc.id
}

resource "railway_variable_collection" "collection" {
  environment_id = railway_environment.dev.id
  service_id     = railway_service.svc.id
  variables = [
    { name = "PORT", value = "8080" },
    { name = "LOG_LEVEL", value = "info" },
  ]
}

# ---- Volumes ----
resource "railway_volume" "data" {
  project_id     = railway_project.acc.id
  service_id     = railway_service.svc.id
  environment_id = railway_environment.dev.id
  mount_path     = "/data"
  name           = "dev-data"
}

resource "railway_volume_backup_schedule" "daily" {
  volume_instance_id = railway_volume.data.volume_instance_id
  kinds              = ["DAILY"]
}

# ---- Networking / edges ----
resource "railway_service_domain" "auto" {
  service_id     = railway_service.svc.id
  environment_id = railway_environment.dev.id
}

resource "railway_tcp_proxy" "redis" {
  application_port = 6379
  environment_id   = railway_environment.dev.id
  service_id       = railway_service.svc.id
}

resource "railway_private_network" "internal" {
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
  name           = "internal"
}

resource "railway_private_network_endpoint" "svc_endpoint" {
  private_network_id = railway_private_network.internal.id
  service_id         = railway_service.svc.id
  environment_id     = railway_environment.dev.id
  service_name       = railway_service.svc.name
}

resource "railway_egress_gateway" "egress" {
  service_id     = railway_service.svc.id
  environment_id = railway_environment.dev.id
}

# ---- Access / notifications ----
resource "railway_project_token" "ci" {
  name           = "ci-token"
  project_id     = railway_project.acc.id
  environment_id = railway_environment.dev.id
}

resource "railway_notification_rule" "deploys" {
  workspace_id = var.workspace_id
  project_id   = railway_project.acc.id
  event_types  = ["deployment.completed"]
  severities   = ["INFO"]
  channel_configs = [
    jsonencode({
      type = "webhook"
      url  = "https://example.com/railway-webhook-target"
    }),
  ]
}
