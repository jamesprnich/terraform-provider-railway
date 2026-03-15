resource "railway_variable" "database_url" {
  name           = "DATABASE_URL"
  value          = "postgres://user:pass@host:5432/db"
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.api.id
}
