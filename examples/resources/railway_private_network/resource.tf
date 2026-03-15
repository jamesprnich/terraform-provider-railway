resource "railway_private_network" "internal" {
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
  name           = "internal"

  # Optional
  # tags = ["backend"]
}
