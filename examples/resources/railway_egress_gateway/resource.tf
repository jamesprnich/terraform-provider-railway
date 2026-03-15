resource "railway_egress_gateway" "api" {
  service_id     = railway_service.api.id
  environment_id = railway_project.example.default_environment.id

  # Optional
  # region = "us-west1"
}
