package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance test is intentionally skipped — notification rules live in the WORKSPACE's
// rule list (even when scoped to a project via project_id). Cleanup behaviour when the
// parent project is deleted is not guaranteed by the Railway API. Requires explicit user
// approval and manual validation in an isolated workspace to enable.
func TestAccNotificationRuleResourceDefault(t *testing.T) {
	t.Skip("workspace-scoped rule list — may persist outside the fixture project. Test manually in an isolated workspace.")
}

func TestNotificationRuleResource_basic(t *testing.T) {
	workspaceId := "ws-abc-123"

	server := newMockGraphQLServer(t, mockFixtures{
		"createNotificationRule": `{"data":{"notificationRuleCreate":{"id":"nr-1","workspaceId":"` + workspaceId + `","projectId":null,"eventTypes":["deployment.completed"],"severities":["INFO"],"ephemeralEnvironments":false,"channels":[{"id":"ch-1","config":{"type":"webhook","url":"https://example.com"}}]}}}`,
		"getNotificationRules":   `{"data":{"notificationRules":[{"id":"nr-1","workspaceId":"` + workspaceId + `","projectId":null,"eventTypes":["deployment.completed"],"severities":["INFO"],"ephemeralEnvironments":false,"channels":[{"id":"ch-1","config":{"type":"webhook","url":"https://example.com"}}]}]}}`,
		"deleteNotificationRule": `{"data":{"notificationRuleDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_notification_rule" "test" {
  workspace_id = "` + workspaceId + `"
  event_types  = ["deployment.completed"]
  severities   = ["INFO"]
  channel_configs = [
    jsonencode({type = "webhook", url = "https://example.com"})
  ]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_notification_rule.test", "id", "nr-1"),
					resource.TestCheckResourceAttr("railway_notification_rule.test", "workspace_id", workspaceId),
					resource.TestCheckResourceAttr("railway_notification_rule.test", "event_types.#", "1"),
					resource.TestCheckResourceAttr("railway_notification_rule.test", "event_types.0", "deployment.completed"),
					resource.TestCheckResourceAttr("railway_notification_rule.test", "severities.#", "1"),
					resource.TestCheckResourceAttr("railway_notification_rule.test", "severities.0", "INFO"),
					resource.TestCheckResourceAttr("railway_notification_rule.test", "channel_configs.#", "1"),
				),
			},
		},
	})
}

func TestNotificationRuleResource_disappears(t *testing.T) {
	workspaceId := "ws-def-456"

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createNotificationRule": `{"data":{"notificationRuleCreate":{"id":"nr-2","workspaceId":"` + workspaceId + `","projectId":null,"eventTypes":["deployment.failed"],"severities":["CRITICAL"],"ephemeralEnvironments":false,"channels":[{"id":"ch-2","config":{"type":"webhook","url":"https://example.com"}}]}}}`,
		"getNotificationRules":   `{"data":{"notificationRules":[{"id":"nr-2","workspaceId":"` + workspaceId + `","projectId":null,"eventTypes":["deployment.failed"],"severities":["CRITICAL"],"ephemeralEnvironments":false,"channels":[{"id":"ch-2","config":{"type":"webhook","url":"https://example.com"}}]}]}}`,
		"deleteNotificationRule": `{"data":{"notificationRuleDelete":true}}`,
	}, "getNotificationRules")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_notification_rule" "test" {
  workspace_id = "` + workspaceId + `"
  event_types  = ["deployment.failed"]
  severities   = ["CRITICAL"]
  channel_configs = [
    jsonencode({type = "webhook", url = "https://example.com"})
  ]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_notification_rule.test", "id", "nr-2"),
					func(s *terraform.State) error {
						disappear()
						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
