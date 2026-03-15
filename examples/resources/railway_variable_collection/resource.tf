resource "railway_variable_collection" "app_config" {
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.api.id

  variables = [
    {
      name  = "DATABASE_URL"
      value = "postgres://user:pass@host:5432/db"
    },
    {
      name  = "REDIS_URL"
      value = "redis://host:6379"
    },
  ]
}
