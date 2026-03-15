# Look up by name (requires project_id)
data "railway_service" "api" {
  name       = "api"
  project_id = railway_project.example.id
}

# Or look up by ID
data "railway_service" "by_id" {
  id = "your-service-id"
}
