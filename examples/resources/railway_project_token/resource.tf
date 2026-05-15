resource "railway_project_token" "ci" {
  name           = "github-actions"
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
}

output "deploy_token" {
  value     = railway_project_token.ci.token
  sensitive = true
}
