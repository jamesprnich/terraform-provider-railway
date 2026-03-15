# Look up by name (requires project_id)
data "railway_environment" "staging" {
  name       = "staging"
  project_id = railway_project.example.id
}

# Or look up by ID
data "railway_environment" "by_id" {
  id = "your-environment-id"
}
