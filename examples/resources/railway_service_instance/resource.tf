resource "railway_service_instance" "backend_dev" {
  service_id     = railway_service.backend.id
  environment_id = railway_environment.dev.id

  source_repo    = "myorg/myapp"
  root_directory = "/backend"
  config_path    = "backend/railway.toml"

  vcpus     = 2
  memory_gb = 0.5

  healthcheck_path = "/health/"
  num_replicas     = 1
}
