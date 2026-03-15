resource "railway_environment" "staging" {
  name       = "staging"
  project_id = railway_project.example.id
}
