resource "railway_deployment_trigger" "api" {
  project_id      = railway_project.example.id
  environment_id  = railway_project.example.default_environment.id
  service_id      = railway_service.api.id
  repository      = "myorg/myapp"
  branch          = "main"
  source_provider = "github"

  # Optional
  # check_suites   = true
  # root_directory = "/backend"
}
