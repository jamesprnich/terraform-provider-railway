# railway_service is an empty per-environment shell. Source, build, deploy,
# and resource-limit configuration lives on `railway_service_instance` (which
# Railway itself scopes per environment).
#
# Under strict_env_scoping (provider default) `environment_id` is required.

resource "railway_service" "api" {
  name           = "api"
  project_id     = railway_project.example.id
  environment_id = railway_environment.dev.id

  # Optional: cosmetic icon shown in the Railway dashboard.
  # icon = "🐹"

  # Optional: attach a persistent volume in the same environment as the service.
  # volume = {
  #   name       = "data"
  #   mount_path = "/data"
  # }
}
