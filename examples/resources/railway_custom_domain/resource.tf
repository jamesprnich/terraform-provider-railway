resource "railway_custom_domain" "api" {
  domain         = "api.example.com"
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.api.id

  # Optional
  # target_port = 8080
}
