resource "railway_bucket" "data" {
  name       = "user-uploads"
  project_id = railway_project.main.id
}
