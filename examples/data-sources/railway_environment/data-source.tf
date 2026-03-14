data "railway_environment" "by_name" {
  name       = "staging"
  project_id = railway_project.example.id
}

data "railway_environment" "by_id" {
  id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
}
