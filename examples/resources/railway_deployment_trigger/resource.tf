resource "railway_deployment_trigger" "example" {
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
  repository     = "myorg/myapp"
  branch         = "main"
  source_provider = "github"
}
