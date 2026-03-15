resource "railway_volume" "data" {
  project_id     = railway_project.example.id
  service_id     = railway_service.postgres.id
  environment_id = railway_project.example.default_environment.id
  mount_path     = "/var/lib/postgresql/data"

  # Optional
  # name = "postgres-data"
}
