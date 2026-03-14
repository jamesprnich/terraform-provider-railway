resource "railway_private_network_endpoint" "example" {
  private_network_id = railway_private_network.example.id
  service_id         = railway_service.example.id
  environment_id     = railway_project.example.default_environment.id
  service_name       = railway_service.example.name
}
