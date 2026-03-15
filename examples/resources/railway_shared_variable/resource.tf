resource "railway_shared_variable" "sentry_dsn" {
  name           = "SENTRY_DSN"
  value          = "https://key@sentry.io/123"
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
}
