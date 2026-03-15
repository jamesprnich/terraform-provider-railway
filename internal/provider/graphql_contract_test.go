package provider

import (
	"encoding/json"
	"testing"
)

// These tests validate that the JSON we expect from Railway's GraphQL API
// correctly deserializes into genqlient's generated response types.
// If a field name, nesting structure, or type is wrong, these tests catch it
// before we hit the live API.

// --- Fragment deserialization tests ---

func TestGraphQLContract_ProjectWebhookFragment(t *testing.T) {
	t.Parallel()

	raw := `{"id":"wh-1","url":"https://example.com","projectId":"proj-1","filters":["deploy.completed"],"lastStatus":200}`

	var fragment ProjectWebhook
	if err := json.Unmarshal([]byte(raw), &fragment); err != nil {
		t.Fatalf("failed to deserialize ProjectWebhook: %s", err)
	}

	assertEqual(t, "Id", fragment.Id, "wh-1")
	assertEqual(t, "Url", fragment.Url, "https://example.com")
	assertEqual(t, "ProjectId", fragment.ProjectId, "proj-1")
	assertIntEqual(t, "LastStatus", fragment.LastStatus, 200)
	assertSliceEqual(t, "Filters", fragment.Filters, []string{"deploy.completed"})
}

func TestGraphQLContract_EgressGatewayFragment(t *testing.T) {
	t.Parallel()

	raw := `{"ipv4":"1.2.3.4","region":"us-west-2"}`

	var fragment EgressGateway
	if err := json.Unmarshal([]byte(raw), &fragment); err != nil {
		t.Fatalf("failed to deserialize EgressGateway: %s", err)
	}

	assertEqual(t, "Ipv4", fragment.Ipv4, "1.2.3.4")
	assertEqual(t, "Region", fragment.Region, "us-west-2")
}

func TestGraphQLContract_PrivateNetworkFieldsFragment(t *testing.T) {
	t.Parallel()

	raw := `{"publicId":"pn-1","projectId":"proj-1","environmentId":"env-1","name":"my-net","dnsName":"my-net.internal","networkId":42,"tags":["web"]}`

	var fragment PrivateNetworkFields
	if err := json.Unmarshal([]byte(raw), &fragment); err != nil {
		t.Fatalf("failed to deserialize PrivateNetworkFields: %s", err)
	}

	assertEqual(t, "PublicId", fragment.PublicId, "pn-1")
	assertEqual(t, "ProjectId", fragment.ProjectId, "proj-1")
	assertEqual(t, "EnvironmentId", fragment.EnvironmentId, "env-1")
	assertEqual(t, "Name", fragment.Name, "my-net")
	assertEqual(t, "DnsName", fragment.DnsName, "my-net.internal")
	assertInt64Equal(t, "NetworkId", fragment.NetworkId, 42)
	assertSliceEqual(t, "Tags", fragment.Tags, []string{"web"})
}

func TestGraphQLContract_PrivateNetworkEndpointFieldsFragment(t *testing.T) {
	t.Parallel()

	raw := `{"publicId":"pne-1","dnsName":"svc.internal","privateIps":["10.0.0.1","10.0.0.2"],"serviceInstanceId":"si-1","tags":["api"]}`

	var fragment PrivateNetworkEndpointFields
	if err := json.Unmarshal([]byte(raw), &fragment); err != nil {
		t.Fatalf("failed to deserialize PrivateNetworkEndpointFields: %s", err)
	}

	assertEqual(t, "PublicId", fragment.PublicId, "pne-1")
	assertEqual(t, "DnsName", fragment.DnsName, "svc.internal")
	assertEqual(t, "ServiceInstanceId", fragment.ServiceInstanceId, "si-1")
	assertSliceEqual(t, "PrivateIps", fragment.PrivateIps, []string{"10.0.0.1", "10.0.0.2"})
	assertSliceEqual(t, "Tags", fragment.Tags, []string{"api"})
}

func TestGraphQLContract_DeploymentTriggerFragment(t *testing.T) {
	t.Parallel()

	raw := `{"id":"dt-1","branch":"main","checkSuites":true,"environmentId":"env-1","projectId":"proj-1","provider":"github","repository":"owner/repo","serviceId":"svc-1"}`

	var fragment DeploymentTrigger
	if err := json.Unmarshal([]byte(raw), &fragment); err != nil {
		t.Fatalf("failed to deserialize DeploymentTrigger: %s", err)
	}

	assertEqual(t, "Id", fragment.Id, "dt-1")
	assertEqual(t, "Branch", fragment.Branch, "main")
	assertBoolEqual(t, "CheckSuites", fragment.CheckSuites, true)
	assertEqual(t, "EnvironmentId", fragment.EnvironmentId, "env-1")
	assertEqual(t, "ProjectId", fragment.ProjectId, "proj-1")
	assertEqual(t, "Provider", fragment.Provider, "github")
	assertEqual(t, "Repository", fragment.Repository, "owner/repo")
	assertEqual(t, "ServiceId", fragment.ServiceId, "svc-1")
}

func TestGraphQLContract_DeploymentTriggerFragment_nullableServiceId(t *testing.T) {
	t.Parallel()

	// serviceId is nullable in the schema (String, not String!) — genqlient maps null to ""
	raw := `{"id":"dt-2","branch":"main","checkSuites":false,"environmentId":"env-1","projectId":"proj-1","provider":"github","repository":"owner/repo","serviceId":null}`

	var fragment DeploymentTrigger
	if err := json.Unmarshal([]byte(raw), &fragment); err != nil {
		t.Fatalf("failed to deserialize DeploymentTrigger with null serviceId: %s", err)
	}

	assertEqual(t, "ServiceId", fragment.ServiceId, "")
}

func TestGraphQLContract_VolumeInstanceBackupScheduleFragment(t *testing.T) {
	t.Parallel()

	raw := `{"id":"vbs-1","kind":"ON_DEMAND","cron":"0 0 * * *","name":"daily-backup","retentionSeconds":86400}`

	var fragment VolumeInstanceBackupSchedule
	if err := json.Unmarshal([]byte(raw), &fragment); err != nil {
		t.Fatalf("failed to deserialize VolumeInstanceBackupSchedule: %s", err)
	}

	assertEqual(t, "Id", fragment.Id, "vbs-1")
	assertEqual(t, "Kind", string(fragment.Kind), "ON_DEMAND")
	assertEqual(t, "Cron", fragment.Cron, "0 0 * * *")
	assertEqual(t, "Name", fragment.Name, "daily-backup")
	assertIntEqual(t, "RetentionSeconds", fragment.RetentionSeconds, 86400)
}

// --- Full response deserialization tests ---
// These test the complete response wrapper structure including edges/nodes.

func TestGraphQLContract_getWebhooksResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"webhooks":{"edges":[{"node":{"id":"wh-1","url":"https://example.com","projectId":"proj-1","filters":["deploy.completed"],"lastStatus":200}}]}}}`

	var envelope struct {
		Data getWebhooksResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize getWebhooksResponse: %s", err)
	}

	edges := envelope.Data.Webhooks.Edges
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	node := edges[0].Node
	assertEqual(t, "Id", node.Id, "wh-1")
	assertEqual(t, "Url", node.Url, "https://example.com")
	assertEqual(t, "ProjectId", node.ProjectId, "proj-1")
}

func TestGraphQLContract_createWebhookResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"webhookCreate":{"id":"wh-1","url":"https://example.com","projectId":"proj-1","filters":[],"lastStatus":0}}}`

	var envelope struct {
		Data createWebhookResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize createWebhookResponse: %s", err)
	}

	webhook := envelope.Data.WebhookCreate
	assertEqual(t, "Id", webhook.Id, "wh-1")
	assertEqual(t, "Url", webhook.Url, "https://example.com")
}

func TestGraphQLContract_getEgressGatewaysResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"egressGateways":[{"ipv4":"1.2.3.4","region":"us-west-2"},{"ipv4":"5.6.7.8","region":"us-west-2"}]}}`

	var envelope struct {
		Data getEgressGatewaysResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize getEgressGatewaysResponse: %s", err)
	}

	gateways := envelope.Data.EgressGateways
	if len(gateways) != 2 {
		t.Fatalf("expected 2 gateways, got %d", len(gateways))
	}

	assertEqual(t, "Ipv4[0]", gateways[0].Ipv4, "1.2.3.4")
	assertEqual(t, "Ipv4[1]", gateways[1].Ipv4, "5.6.7.8")
}

func TestGraphQLContract_createEgressGatewayResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"egressGatewayAssociationCreate":[{"ipv4":"1.2.3.4","region":"us-west-2"}]}}`

	var envelope struct {
		Data createEgressGatewayResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize createEgressGatewayResponse: %s", err)
	}

	gateways := envelope.Data.EgressGatewayAssociationCreate
	if len(gateways) != 1 {
		t.Fatalf("expected 1 gateway, got %d", len(gateways))
	}

	assertEqual(t, "Ipv4", gateways[0].Ipv4, "1.2.3.4")
}

func TestGraphQLContract_getPrivateNetworksResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"privateNetworks":[{"publicId":"pn-1","projectId":"proj-1","environmentId":"env-1","name":"default","dnsName":"default.internal","networkId":100,"tags":["web"]}]}}`

	var envelope struct {
		Data getPrivateNetworksResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize getPrivateNetworksResponse: %s", err)
	}

	networks := envelope.Data.PrivateNetworks
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}

	assertEqual(t, "PublicId", networks[0].PublicId, "pn-1")
	assertEqual(t, "Name", networks[0].Name, "default")
	assertInt64Equal(t, "NetworkId", networks[0].NetworkId, 100)
}

func TestGraphQLContract_getPrivateNetworkEndpointResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"privateNetworkEndpoint":{"publicId":"pne-1","dnsName":"svc.internal","privateIps":["10.0.0.1"],"serviceInstanceId":"si-1","tags":[]}}}`

	var envelope struct {
		Data getPrivateNetworkEndpointResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize getPrivateNetworkEndpointResponse: %s", err)
	}

	endpoint := envelope.Data.PrivateNetworkEndpoint
	assertEqual(t, "PublicId", endpoint.PublicId, "pne-1")
	assertEqual(t, "DnsName", endpoint.DnsName, "svc.internal")
}

func TestGraphQLContract_getDeploymentTriggersResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"deploymentTriggers":{"edges":[{"node":{"id":"dt-1","branch":"main","checkSuites":true,"environmentId":"env-1","projectId":"proj-1","provider":"github","repository":"owner/repo","serviceId":"svc-1"}}]}}}`

	var envelope struct {
		Data getDeploymentTriggersResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize getDeploymentTriggersResponse: %s", err)
	}

	edges := envelope.Data.DeploymentTriggers.Edges
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	node := edges[0].Node
	assertEqual(t, "Id", node.Id, "dt-1")
	assertEqual(t, "Branch", node.Branch, "main")
	assertEqual(t, "Provider", node.Provider, "github")
}

func TestGraphQLContract_createDeploymentTriggerResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"deploymentTriggerCreate":{"id":"dt-1","branch":"main","checkSuites":false,"environmentId":"env-1","projectId":"proj-1","provider":"github","repository":"owner/repo","serviceId":"svc-1"}}}`

	var envelope struct {
		Data createDeploymentTriggerResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize createDeploymentTriggerResponse: %s", err)
	}

	trigger := envelope.Data.DeploymentTriggerCreate
	assertEqual(t, "Id", trigger.Id, "dt-1")
	assertEqual(t, "Repository", trigger.Repository, "owner/repo")
	assertEqual(t, "ServiceId", trigger.ServiceId, "svc-1")
}

func TestGraphQLContract_getVolumeInstanceBackupSchedulesResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"volumeInstanceBackupScheduleList":[{"id":"vbs-1","kind":"ON_DEMAND","cron":"0 0 * * *","name":"daily","retentionSeconds":86400}]}}`

	var envelope struct {
		Data getVolumeInstanceBackupSchedulesResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize getVolumeInstanceBackupSchedulesResponse: %s", err)
	}

	schedules := envelope.Data.VolumeInstanceBackupScheduleList
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}

	assertEqual(t, "Id", schedules[0].Id, "vbs-1")
	assertEqual(t, "Kind", string(schedules[0].Kind), "ON_DEMAND")
}

func TestGraphQLContract_listProjectsResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"projects":{"edges":[{"node":{"id":"proj-1","name":"my-project","description":"desc"}}]}}}`

	var envelope struct {
		Data listProjectsResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize listProjectsResponse: %s", err)
	}

	edges := envelope.Data.Projects.Edges
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	assertEqual(t, "Id", edges[0].Node.Id, "proj-1")
	assertEqual(t, "Name", edges[0].Node.Name, "my-project")
}

func TestGraphQLContract_getProjectServicesResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"project":{"services":{"edges":[{"node":{"id":"svc-1","name":"web"}}]}}}}`

	var envelope struct {
		Data getProjectServicesResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize getProjectServicesResponse: %s", err)
	}

	edges := envelope.Data.Project.Services.Edges
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	assertEqual(t, "Id", edges[0].Node.Id, "svc-1")
	assertEqual(t, "Name", edges[0].Node.Name, "web")
}

// --- Input type serialization tests ---
// Verify input structs serialize to the JSON the API expects.

func TestGraphQLContract_WebhookCreateInput_serialization(t *testing.T) {
	t.Parallel()

	input := WebhookCreateInput{
		ProjectId: "proj-1",
		Url:       "https://example.com",
		Filters:   []string{"deploy.completed"},
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal WebhookCreateInput: %s", err)
	}

	var m map[string]interface{}
	json.Unmarshal(b, &m)

	assertEqual(t, "projectId", m["projectId"].(string), "proj-1")
	assertEqual(t, "url", m["url"].(string), "https://example.com")

	filters := m["filters"].([]interface{})
	if len(filters) != 1 || filters[0].(string) != "deploy.completed" {
		t.Errorf("filters mismatch: got %v", filters)
	}
}

func TestGraphQLContract_EgressGatewayCreateInput_serialization(t *testing.T) {
	t.Parallel()

	region := "us-west-2"
	input := EgressGatewayCreateInput{
		ServiceId:     "svc-1",
		EnvironmentId: "env-1",
		Region:        &region,
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal EgressGatewayCreateInput: %s", err)
	}

	var m map[string]interface{}
	json.Unmarshal(b, &m)

	assertEqual(t, "serviceId", m["serviceId"].(string), "svc-1")
	assertEqual(t, "environmentId", m["environmentId"].(string), "env-1")
	assertEqual(t, "region", m["region"].(string), "us-west-2")

	// Verify omitempty: when Region is nil, it should be omitted from JSON
	inputNoRegion := EgressGatewayCreateInput{
		ServiceId:     "svc-2",
		EnvironmentId: "env-2",
	}
	b2, err := json.Marshal(inputNoRegion)
	if err != nil {
		t.Fatalf("failed to marshal EgressGatewayCreateInput without region: %s", err)
	}
	var m2 map[string]interface{}
	json.Unmarshal(b2, &m2)
	if _, exists := m2["region"]; exists {
		t.Errorf("expected region to be omitted when nil, but got: %v", m2["region"])
	}
}

func TestGraphQLContract_DeploymentTriggerCreateInput_serialization(t *testing.T) {
	t.Parallel()

	checkSuites := true
	rootDir := "/app"
	input := DeploymentTriggerCreateInput{
		Branch:        "main",
		EnvironmentId: "env-1",
		ProjectId:     "proj-1",
		Provider:      "github",
		Repository:    "owner/repo",
		ServiceId:     "svc-1",
		CheckSuites:   &checkSuites,
		RootDirectory: &rootDir,
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal DeploymentTriggerCreateInput: %s", err)
	}

	var m map[string]interface{}
	json.Unmarshal(b, &m)

	assertEqual(t, "branch", m["branch"].(string), "main")
	assertEqual(t, "provider", m["provider"].(string), "github")
	assertEqual(t, "repository", m["repository"].(string), "owner/repo")
	assertEqual(t, "serviceId", m["serviceId"].(string), "svc-1")
	assertBoolEqual(t, "checkSuites", m["checkSuites"].(bool), true)
	assertEqual(t, "rootDirectory", m["rootDirectory"].(string), "/app")
}

func TestGraphQLContract_DeploymentTriggerCreateInput_omitsNulls(t *testing.T) {
	t.Parallel()

	// When optional fields are nil, they should be omitted from JSON
	input := DeploymentTriggerCreateInput{
		Branch:        "main",
		EnvironmentId: "env-1",
		ProjectId:     "proj-1",
		Provider:      "github",
		Repository:    "owner/repo",
		ServiceId:     "svc-1",
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal DeploymentTriggerCreateInput: %s", err)
	}

	var m map[string]interface{}
	json.Unmarshal(b, &m)

	if _, ok := m["checkSuites"]; ok {
		t.Error("checkSuites should be omitted when nil")
	}
	if _, ok := m["rootDirectory"]; ok {
		t.Error("rootDirectory should be omitted when nil")
	}
}

func TestGraphQLContract_DeploymentTriggerUpdateInput_serialization(t *testing.T) {
	t.Parallel()

	branch := "develop"
	input := DeploymentTriggerUpdateInput{
		Branch: &branch,
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal DeploymentTriggerUpdateInput: %s", err)
	}

	var m map[string]interface{}
	json.Unmarshal(b, &m)

	assertEqual(t, "branch", m["branch"].(string), "develop")

	// Other fields should be omitted
	if _, ok := m["checkSuites"]; ok {
		t.Error("checkSuites should be omitted when nil")
	}
}

func TestGraphQLContract_PrivateNetworkEndpointCreateOrGetInput_serialization(t *testing.T) {
	t.Parallel()

	input := PrivateNetworkEndpointCreateOrGetInput{
		EnvironmentId:    "env-1",
		PrivateNetworkId: "pn-1",
		ServiceId:        "svc-1",
		ServiceName:      "web",
		Tags:             []string{"api"},
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal PrivateNetworkEndpointCreateOrGetInput: %s", err)
	}

	var m map[string]interface{}
	json.Unmarshal(b, &m)

	assertEqual(t, "environmentId", m["environmentId"].(string), "env-1")
	assertEqual(t, "privateNetworkId", m["privateNetworkId"].(string), "pn-1")
	assertEqual(t, "serviceId", m["serviceId"].(string), "svc-1")
	assertEqual(t, "serviceName", m["serviceName"].(string), "web")
}

// --- Mutation response field name tests ---
// Verify the top-level mutation response field names match the API.

func TestGraphQLContract_mutationResponseFieldNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		validate func(t *testing.T, raw []byte)
	}{
		{
			name: "webhookDelete returns boolean at webhookDelete key",
			json: `{"data":{"webhookDelete":true}}`,
			validate: func(t *testing.T, raw []byte) {
				var envelope struct {
					Data deleteWebhookResponse `json:"data"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatalf("deserialize error: %s", err)
				}
				assertBoolEqual(t, "WebhookDelete", envelope.Data.WebhookDelete, true)
			},
		},
		{
			name: "egressGatewayAssociationsClear returns boolean",
			json: `{"data":{"egressGatewayAssociationsClear":true}}`,
			validate: func(t *testing.T, raw []byte) {
				var envelope struct {
					Data clearEgressGatewaysResponse `json:"data"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatalf("deserialize error: %s", err)
				}
				assertBoolEqual(t, "EgressGatewayAssociationsClear", envelope.Data.EgressGatewayAssociationsClear, true)
			},
		},
		{
			name: "privateNetworksForEnvironmentDelete returns boolean",
			json: `{"data":{"privateNetworksForEnvironmentDelete":true}}`,
			validate: func(t *testing.T, raw []byte) {
				var envelope struct {
					Data deletePrivateNetworksForEnvironmentResponse `json:"data"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatalf("deserialize error: %s", err)
				}
				assertBoolEqual(t, "PrivateNetworksForEnvironmentDelete", envelope.Data.PrivateNetworksForEnvironmentDelete, true)
			},
		},
		{
			name: "privateNetworkEndpointDelete returns boolean",
			json: `{"data":{"privateNetworkEndpointDelete":true}}`,
			validate: func(t *testing.T, raw []byte) {
				var envelope struct {
					Data deletePrivateNetworkEndpointResponse `json:"data"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatalf("deserialize error: %s", err)
				}
				assertBoolEqual(t, "PrivateNetworkEndpointDelete", envelope.Data.PrivateNetworkEndpointDelete, true)
			},
		},
		{
			name: "privateNetworkEndpointRename returns boolean",
			json: `{"data":{"privateNetworkEndpointRename":true}}`,
			validate: func(t *testing.T, raw []byte) {
				var envelope struct {
					Data renamePrivateNetworkEndpointResponse `json:"data"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatalf("deserialize error: %s", err)
				}
				assertBoolEqual(t, "PrivateNetworkEndpointRename", envelope.Data.PrivateNetworkEndpointRename, true)
			},
		},
		{
			name: "deploymentTriggerDelete returns boolean",
			json: `{"data":{"deploymentTriggerDelete":true}}`,
			validate: func(t *testing.T, raw []byte) {
				var envelope struct {
					Data deleteDeploymentTriggerResponse `json:"data"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatalf("deserialize error: %s", err)
				}
				assertBoolEqual(t, "DeploymentTriggerDelete", envelope.Data.DeploymentTriggerDelete, true)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.validate(t, []byte(tc.json))
		})
	}
}

// --- Modified resource response tests ---

func TestGraphQLContract_renameEnvironmentResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"environmentRename":{"id":"env-1","name":"staging","projectId":"proj-1"}}}`

	var envelope struct {
		Data renameEnvironmentResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize renameEnvironmentResponse: %s", err)
	}

	env := envelope.Data.EnvironmentRename
	assertEqual(t, "Id", env.Id, "env-1")
	assertEqual(t, "Name", env.Name, "staging")
}

func TestGraphQLContract_updateCustomDomainResponse(t *testing.T) {
	t.Parallel()

	raw := `{"data":{"customDomainUpdate":true}}`

	var envelope struct {
		Data updateCustomDomainResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("failed to deserialize updateCustomDomainResponse: %s", err)
	}

	assertBoolEqual(t, "CustomDomainUpdate", envelope.Data.CustomDomainUpdate, true)
}

// --- Test helpers ---

func assertEqual(t *testing.T, field string, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", field, got, want)
	}
}

func assertIntEqual(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %d, want %d", field, got, want)
	}
}

func assertInt64Equal(t *testing.T, field string, got, want int64) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %d, want %d", field, got, want)
	}
}

func assertBoolEqual(t *testing.T, field string, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

func assertSliceEqual(t *testing.T, field string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: length mismatch: got %d, want %d", field, len(got), len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q", field, i, got[i], want[i])
		}
	}
}
