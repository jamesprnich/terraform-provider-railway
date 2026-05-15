---
page_title: "railway_notification_rule Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway notification rule.
---

# railway_notification_rule (Resource)

Railway notification rule. Sends notifications (via webhook, Slack, email, or other channels) when events of the configured types occur.

~> **Migration from `railway_webhook`:** This resource replaces the deprecated `railway_webhook` resource that was removed in v0.9.0 when Railway removed the `webhookCreate/Update/Delete` mutations from its API. Webhooks are now one channel type among many. Existing `railway_webhook.X` resources in state should be removed (`tofu state rm`) and re-created as `railway_notification_rule.X`.

## Example Usage

```terraform
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
```

## Schema

### Required

- `workspace_id` (String) Identifier of the workspace the rule belongs to.
- `event_types` (List of String) Event types that trigger the rule (e.g. `deployment.completed`, `deployment.failed`).
- `channel_configs` (List of String) List of channel configurations as JSON strings. Each entry describes one delivery channel (webhook, Slack, email, etc.). Refer to the Railway API documentation for the supported channel shapes.

### Optional

- `project_id` (String) Identifier of the project the rule scopes to. When omitted, the rule applies to the entire workspace.
- `severities` (List of String) Severity levels to notify on. Each value must be one of `CRITICAL`, `INFO`, `NOTICE`, `WARNING`.
- `ephemeral_environments` (Boolean) Whether to notify for events on ephemeral (PR) environments.

### Read-Only

- `id` (String) Identifier of the notification rule.

## Import

Import is supported using the following syntax:

```shell
tofu import railway_notification_rule.deploy_alerts <workspace_id>:<rule_id>
```
