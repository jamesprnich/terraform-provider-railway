resource "railway_webhook" "example" {
  project_id = railway_project.example.id
  url        = "https://example.com/webhook"
}
