data "railway_service" "by_name" {
  name       = "api"
  project_id = railway_project.example.id
}

data "railway_service" "by_id" {
  id = "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"
}
