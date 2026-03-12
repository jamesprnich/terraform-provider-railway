resource "railway_volume" "postgres_data" {
  project_id     = railway_project.example.id
  service_id     = railway_service.postgres.id
  environment_id = railway_environment.dev.id
  mount_path     = "/var/lib/postgresql/data"
  name           = "postgres-data"
}
