resource "railway_service" "api" {
  name       = "api"
  project_id = railway_project.example.id

  # Optional: deploy from a Docker image
  # source_image = "nginx:1.27-alpine"

  # Optional: deploy from a GitHub repo (requires both)
  # source_repo        = "myorg/myapp"
  # source_repo_branch = "main"
}
