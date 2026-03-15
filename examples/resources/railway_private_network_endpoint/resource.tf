resource "railway_private_network_endpoint" "api" {
  private_network_id = railway_private_network.internal.id
  service_id         = railway_service.api.id
  environment_id     = railway_project.example.default_environment.id
  service_name       = railway_service.api.name

  # Optional
  # dns_name = "api-internal"
  # tags     = ["backend"]
}
