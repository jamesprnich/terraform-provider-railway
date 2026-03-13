resource "railway_service_domain" "api" {
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}
