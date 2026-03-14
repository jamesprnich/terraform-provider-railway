resource "railway_egress_gateway" "example" {
  service_id     = railway_service.example.id
  environment_id = railway_project.example.default_environment.id
}
