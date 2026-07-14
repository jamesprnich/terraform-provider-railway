# railway_service_instance holds the per-environment source, build, deploy,
# and resource-limit configuration for a service. Under Railway's own data
# model, source is a per-environment concern — a service in `dev` and the
# same service in `prd` can point at different repos, images, or branches.
#
# In v0.11.0 all source-related fields moved off `railway_service` (where
# their underlying Railway mutation was env-less and leaked across every
# non-fork environment) and live exclusively here.

resource "railway_service_instance" "api" {
  service_id     = railway_service.api.id
  environment_id = railway_environment.dev.id

  # Source — pick ONE: source_image (Docker) OR source_repo + source_repo_branch (GitHub).
  source_repo        = "myorg/myapp"
  source_repo_branch = "main"
  root_directory     = "backend"
  # config_path      = "railway.json"

  # Docker image alternative:
  # source_image = "myorg/myapp:latest"

  # Private registry credentials (required when source_image references a private image).
  # registry_credentials = {
  #   username = "myuser"
  #   password = var.registry_token   # mark as sensitive in your vars
  # }

  # Build and deploy settings.
  # build_command    = "npm run build"
  # start_command    = "npm start"
  # healthcheck_path = "/health"
  # num_replicas     = 1
  # region           = "us-west1"
  vcpus     = 2
  memory_gb = 0.5

  # Deploy behavior.
  # sleep_application          = true
  # overlap_seconds            = 5
  # draining_seconds           = 10
  # healthcheck_timeout        = 300
  # restart_policy_type        = "ON_FAILURE"
  # restart_policy_max_retries = 3
  # builder                    = "RAILPACK"
  # watch_patterns             = ["backend/**"]
  # pre_deploy_command         = "python manage.py migrate"

  # Cron mode — only allowed when num_replicas across all regions is 1.
  # cron_schedule = "0 3 * * *"
}
