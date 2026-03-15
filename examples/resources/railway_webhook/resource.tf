resource "railway_webhook" "notifications" {
  project_id = railway_project.example.id
  url        = "https://example.com/webhook"

  # Optional: filter to specific events
  # filters = ["DEPLOY"]
}
