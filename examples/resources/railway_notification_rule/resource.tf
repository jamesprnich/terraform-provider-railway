resource "railway_notification_rule" "deploy_alerts" {
  workspace_id = var.railway_workspace_id
  project_id   = railway_project.main.id
  event_types  = ["deployment.completed", "deployment.failed"]
  severities   = ["CRITICAL", "WARNING"]
  channel_configs = [
    jsonencode({
      type = "webhook"
      url  = "https://example.com/railway-hook"
    }),
  ]
}
