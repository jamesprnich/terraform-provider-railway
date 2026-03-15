package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccNewClient creates a graphql.Client for use in acceptance test helpers
// that need to interact with the Railway API directly (e.g., disappears tests).
func testAccNewClient() graphql.Client {
	token := os.Getenv("RAILWAY_TOKEN")
	httpClient := http.Client{
		Transport: &authedTransport{
			token:   token,
			wrapped: http.DefaultTransport,
		},
	}
	return graphql.NewClient(defaultAPIURL, &httpClient)
}

// testAccWaitUntilGone polls the check function until it returns a not-found error,
// confirming the resource has been fully removed from Railway's API (eventual consistency).
func testAccWaitUntilGone(check func() error) error {
	ctx := context.Background()
	return retry.RetryContext(ctx, 30*time.Second, func() *retry.RetryError {
		err := check()
		if err == nil {
			// Resource still exists — keep polling
			return retry.RetryableError(fmt.Errorf("resource still exists"))
		}
		if isNotFound(err) {
			// Resource is gone
			return nil
		}
		// Unexpected error
		return retry.NonRetryableError(err)
	})
}

// testAccCheckProjectDisappears deletes the project externally via the API.
func testAccCheckProjectDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteProject(context.Background(), client, rs.Primary.ID)
		return err
	}
}

// testAccCheckEnvironmentDisappears deletes the environment externally via the API.
func testAccCheckEnvironmentDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		resp, err := deleteEnvironment(context.Background(), client, rs.Primary.ID)
		if err != nil {
			return err
		}
		if !resp.EnvironmentDelete {
			return fmt.Errorf("environmentDelete returned false for environment %s", rs.Primary.ID)
		}
		// Poll the project's environment list until the environment is gone.
		// The individual environment(id:) query can return stale data longer
		// than the environments(projectId:) list query.
		projectId := rs.Primary.Attributes["project_id"]
		envId := rs.Primary.ID
		return testAccWaitUntilGone(func() error {
			response, err := getEnvironments(context.Background(), client, projectId)
			if err != nil {
				return err
			}
			for _, edge := range response.Environments.Edges {
				if edge.Node.Environment.Id == envId {
					return nil // still in list
				}
			}
			return &NotFoundError{ResourceType: "environment", Id: envId}
		})
	}
}

// testAccCheckServiceDisappears deletes the service externally via the API.
func testAccCheckServiceDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteService(context.Background(), client, rs.Primary.ID)
		return err
	}
}

// testAccCheckVariableDisappears deletes the variable externally via the API.
func testAccCheckVariableDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		serviceId := rs.Primary.Attributes["service_id"]
		client := testAccNewClient()
		_, err := deleteVariable(context.Background(), client, VariableDeleteInput{
			Name:          rs.Primary.Attributes["name"],
			EnvironmentId: rs.Primary.Attributes["environment_id"],
			ProjectId:     rs.Primary.Attributes["project_id"],
			ServiceId:     &serviceId,
		})
		return err
	}
}

// testAccCheckVariableCollectionDisappears deletes all variables in the collection externally.
func testAccCheckVariableCollectionDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		serviceId := rs.Primary.Attributes["service_id"]
		environmentId := rs.Primary.Attributes["environment_id"]
		projectId := rs.Primary.Attributes["project_id"]

		// Get variable names from the variables list attributes
		var varNames []string
		for i := 0; ; i++ {
			name, ok := rs.Primary.Attributes[fmt.Sprintf("variables.%d.name", i)]
			if !ok {
				break
			}
			varNames = append(varNames, name)
		}
		if len(varNames) == 0 {
			return fmt.Errorf("no variable names found in state for %s", resourceName)
		}

		for _, name := range varNames {
			_, err := deleteVariable(context.Background(), client, VariableDeleteInput{
				Name:          name,
				EnvironmentId: environmentId,
				ProjectId:     projectId,
				ServiceId:     &serviceId,
			})
			if err != nil {
				return fmt.Errorf("failed to delete variable %s: %w", name, err)
			}
		}
		return nil
	}
}

// testAccCheckSharedVariableDisappears deletes the shared variable externally via the API.
func testAccCheckSharedVariableDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteVariable(context.Background(), client, VariableDeleteInput{
			Name:          rs.Primary.Attributes["name"],
			EnvironmentId: rs.Primary.Attributes["environment_id"],
			ProjectId:     rs.Primary.Attributes["project_id"],
		})
		return err
	}
}

// testAccCheckServiceDomainDisappears deletes the service domain externally via the API.
func testAccCheckServiceDomainDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteServiceDomain(context.Background(), client, rs.Primary.ID)
		if err != nil {
			return err
		}
		// Poll until domain is confirmed gone
		projectId := rs.Primary.Attributes["project_id"]
		envId := rs.Primary.Attributes["environment_id"]
		svcId := rs.Primary.Attributes["service_id"]
		domainId := rs.Primary.ID
		return testAccWaitUntilGone(func() error {
			_, err := findServiceDomainById(context.Background(), client, projectId, envId, svcId, domainId)
			return err
		})
	}
}

// testAccCheckCustomDomainDisappears deletes the custom domain externally via the API.
func testAccCheckCustomDomainDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteCustomDomain(context.Background(), client, rs.Primary.ID)
		if err != nil {
			return err
		}
		// Poll until domain is confirmed gone
		projectId := rs.Primary.Attributes["project_id"]
		envId := rs.Primary.Attributes["environment_id"]
		svcId := rs.Primary.Attributes["service_id"]
		domainId := rs.Primary.ID
		return testAccWaitUntilGone(func() error {
			response, err := listCustomDomains(context.Background(), client, envId, svcId, projectId)
			if err != nil {
				return err
			}
			for _, cd := range response.Domains.CustomDomains {
				if cd.CustomDomain.Id == domainId {
					return nil // still exists
				}
			}
			return &NotFoundError{ResourceType: "custom domain", Id: domainId}
		})
	}
}

// testAccCheckTcpProxyDisappears deletes the TCP proxy externally via the API.
func testAccCheckTcpProxyDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteTcpProxy(context.Background(), client, rs.Primary.ID)
		if err != nil {
			return err
		}
		// Poll until proxy is confirmed gone
		envId := rs.Primary.Attributes["environment_id"]
		svcId := rs.Primary.Attributes["service_id"]
		proxyId := rs.Primary.ID
		return testAccWaitUntilGone(func() error {
			response, err := getTcpProxy(context.Background(), client, envId, svcId)
			if err != nil {
				return err
			}
			for _, proxy := range response.TcpProxies {
				if proxy.Id == proxyId {
					return nil // still exists
				}
			}
			return &NotFoundError{ResourceType: "tcp proxy", Id: proxyId}
		})
	}
}

// testAccCheckWebhookDisappears deletes the webhook externally via the API.
func testAccCheckWebhookDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		ctx := context.Background()
		_, err := deleteWebhook(ctx, client, rs.Primary.ID)
		if err != nil {
			return err
		}
		// Poll until webhook is confirmed gone
		projectId := rs.Primary.Attributes["project_id"]
		webhookId := rs.Primary.ID
		return testAccWaitUntilGone(func() error {
			response, err := getWebhooks(ctx, client, projectId)
			if err != nil {
				return err
			}
			for _, edge := range response.Webhooks.Edges {
				if edge.Node.Id == webhookId {
					return nil // still exists
				}
			}
			return &NotFoundError{ResourceType: "webhook", Id: webhookId}
		})
	}
}

// testAccCheckEgressGatewayDisappears clears the egress gateway externally via the API.
func testAccCheckEgressGatewayDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		ctx := context.Background()
		serviceId := rs.Primary.Attributes["service_id"]
		envId := rs.Primary.Attributes["environment_id"]
		_, err := clearEgressGateways(ctx, client, EgressGatewayServiceTargetInput{
			ServiceId:     serviceId,
			EnvironmentId: envId,
		})
		if err != nil {
			return err
		}
		// Poll until egress gateways are confirmed gone
		return testAccWaitUntilGone(func() error {
			response, err := getEgressGateways(ctx, client, envId, serviceId)
			if err != nil {
				return err
			}
			if len(response.EgressGateways) > 0 {
				return nil // still exists
			}
			return &NotFoundError{ResourceType: "egress gateway", Id: rs.Primary.ID}
		})
	}
}

// testAccCheckDeploymentTriggerDisappears deletes the deployment trigger externally via the API.
func testAccCheckDeploymentTriggerDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteDeploymentTrigger(context.Background(), client, rs.Primary.ID)
		return err
	}
}

// testAccCheckPrivateNetworkDisappears deletes private networks for the environment externally via the API.
func testAccCheckPrivateNetworkDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		ctx := context.Background()
		envId := rs.Primary.Attributes["environment_id"]
		_, err := deletePrivateNetworksForEnvironment(ctx, client, envId)
		if err != nil {
			return err
		}
		// Poll until private networks are confirmed gone
		return testAccWaitUntilGone(func() error {
			response, err := getPrivateNetworks(ctx, client, envId)
			if err != nil {
				return err
			}
			if len(response.PrivateNetworks) > 0 {
				return nil // still exists
			}
			return &NotFoundError{ResourceType: "private network", Id: rs.Primary.ID}
		})
	}
}

// testAccCheckPrivateNetworkEndpointDisappears deletes the private network endpoint externally via the API.
func testAccCheckPrivateNetworkEndpointDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deletePrivateNetworkEndpoint(context.Background(), client, rs.Primary.ID)
		return err
	}
}

// testAccCheckVolumeDisappears deletes the volume externally via the API.
func testAccCheckVolumeDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		_, err := deleteVolume(context.Background(), client, rs.Primary.ID)
		return err
	}
}

// testAccCheckServiceInstanceDisappears deletes the parent service externally via the API.
// Service instances are implicit and cannot be deleted directly — deleting the parent service
// causes the service instance to disappear.
func testAccCheckServiceInstanceDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found in state: %s", resourceName)
		}
		client := testAccNewClient()
		serviceId := rs.Primary.Attributes["service_id"]
		_, err := deleteService(context.Background(), client, serviceId)
		return err
	}
}

// =============================================================================
// CheckDestroy functions — verify resources are actually deleted after destroy
// =============================================================================

// testAccCheckProjectDestroy verifies all railway_project resources have been deleted.
func testAccCheckProjectDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_project" {
			continue
		}
		_, err := getProject(ctx, client, rs.Primary.ID)
		if isNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("railway_project %s still exists after destroy", rs.Primary.ID)
	}
	return nil
}

// testAccCheckEnvironmentDestroy verifies all railway_environment resources have been deleted.
func testAccCheckEnvironmentDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_environment" {
			continue
		}
		projectId := rs.Primary.Attributes["project_id"]
		response, err := getEnvironments(ctx, client, projectId)
		if isNotFound(err) {
			continue // parent project is gone, so environment is gone
		}
		if err != nil {
			return err
		}
		for _, edge := range response.Environments.Edges {
			if edge.Node.Environment.Id == rs.Primary.ID {
				return fmt.Errorf("railway_environment %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckServiceDestroy verifies all railway_service resources have been deleted.
// Uses the project services list query (not getService by ID) because Railway's API
// returns stale data on individual resource queries for 30+ seconds after deletion.
// The list endpoint reflects deletions within 1-2 seconds.
func testAccCheckServiceDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_service" {
			continue
		}
		serviceId := rs.Primary.ID
		projectId := rs.Primary.Attributes["project_id"]
		err := testAccWaitUntilGone(func() error {
			response, err := getProjectServices(ctx, client, projectId)
			if isNotFound(err) {
				return &NotFoundError{ResourceType: "service", Id: serviceId}
			}
			if err != nil {
				return err
			}
			for _, edge := range response.Project.Services.Edges {
				if edge.Node.Id == serviceId {
					return nil // still in list
				}
			}
			return &NotFoundError{ResourceType: "service", Id: serviceId}
		})
		if err != nil {
			return fmt.Errorf("railway_service %s still exists after destroy", serviceId)
		}
	}
	return nil
}

// testAccCheckServiceInstanceDestroy is a no-op because service instances are implicit —
// they can't be destroyed, only reset to defaults. The instance always exists as long
// as the parent service exists. The service_instance Delete method resets config but
// doesn't delete the instance.
func testAccCheckServiceInstanceDestroy(s *terraform.State) error {
	return nil
}

// testAccCheckVariableDestroy verifies all railway_variable resources have been deleted.
func testAccCheckVariableDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_variable" {
			continue
		}
		projectId := rs.Primary.Attributes["project_id"]
		environmentId := rs.Primary.Attributes["environment_id"]
		serviceId := rs.Primary.Attributes["service_id"]
		name := rs.Primary.Attributes["name"]
		response, err := getVariables(ctx, client, projectId, environmentId, serviceId)
		if isNotFound(err) {
			continue // parent resource is gone, so variable is gone
		}
		if err != nil {
			return err
		}
		if _, exists := response.Variables[name]; exists {
			return fmt.Errorf("railway_variable %s (name=%s) still exists after destroy", rs.Primary.ID, name)
		}
	}
	return nil
}

// testAccCheckVariableCollectionDestroy verifies all railway_variable_collection resources have been deleted.
func testAccCheckVariableCollectionDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_variable_collection" {
			continue
		}
		projectId := rs.Primary.Attributes["project_id"]
		environmentId := rs.Primary.Attributes["environment_id"]
		serviceId := rs.Primary.Attributes["service_id"]

		// Collect all variable names from the collection
		var varNames []string
		for i := 0; ; i++ {
			name, ok := rs.Primary.Attributes[fmt.Sprintf("variables.%d.name", i)]
			if !ok {
				break
			}
			varNames = append(varNames, name)
		}

		response, err := getVariables(ctx, client, projectId, environmentId, serviceId)
		if isNotFound(err) {
			continue // parent resource is gone, so variables are gone
		}
		if err != nil {
			return err
		}
		for _, name := range varNames {
			if _, exists := response.Variables[name]; exists {
				return fmt.Errorf("railway_variable_collection %s: variable %s still exists after destroy", rs.Primary.ID, name)
			}
		}
	}
	return nil
}

// testAccCheckSharedVariableDestroy verifies all railway_shared_variable resources have been deleted.
func testAccCheckSharedVariableDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_shared_variable" {
			continue
		}
		projectId := rs.Primary.Attributes["project_id"]
		environmentId := rs.Primary.Attributes["environment_id"]
		name := rs.Primary.Attributes["name"]
		response, err := getSharedVariables(ctx, client, projectId, environmentId)
		if isNotFound(err) {
			continue // parent project is gone, so variable is gone
		}
		if err != nil {
			return err
		}
		if _, exists := response.Variables[name]; exists {
			return fmt.Errorf("railway_shared_variable %s (name=%s) still exists after destroy", rs.Primary.ID, name)
		}
	}
	return nil
}

// testAccCheckVolumeDestroy verifies all railway_volume resources have been deleted.
func testAccCheckVolumeDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_volume" {
			continue
		}
		projectId := rs.Primary.Attributes["project_id"]
		response, err := getVolumeInstances(ctx, client, projectId)
		if isNotFound(err) {
			continue // parent project is gone, so volume is gone
		}
		if err != nil {
			return err
		}
		for _, edge := range response.Project.Volumes.Edges {
			if edge.Node.Volume.Id == rs.Primary.ID {
				return fmt.Errorf("railway_volume %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckServiceDomainDestroy verifies all railway_service_domain resources have been deleted.
func testAccCheckServiceDomainDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_service_domain" {
			continue
		}
		environmentId := rs.Primary.Attributes["environment_id"]
		serviceId := rs.Primary.Attributes["service_id"]
		projectId := rs.Primary.Attributes["project_id"]
		response, err := listServiceDomains(ctx, client, environmentId, serviceId, projectId)
		if isNotFound(err) {
			continue // parent service/project is gone, so domain is gone
		}
		if err != nil {
			return err
		}
		for _, sd := range response.Domains.ServiceDomains {
			if sd.ServiceDomain.Id == rs.Primary.ID {
				return fmt.Errorf("railway_service_domain %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckCustomDomainDestroy verifies all railway_custom_domain resources have been deleted.
func testAccCheckCustomDomainDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_custom_domain" {
			continue
		}
		environmentId := rs.Primary.Attributes["environment_id"]
		serviceId := rs.Primary.Attributes["service_id"]
		projectId := rs.Primary.Attributes["project_id"]
		response, err := listCustomDomains(ctx, client, environmentId, serviceId, projectId)
		if isNotFound(err) {
			continue // parent service/project is gone, so domain is gone
		}
		if err != nil {
			return err
		}
		for _, cd := range response.Domains.CustomDomains {
			if cd.CustomDomain.Id == rs.Primary.ID {
				return fmt.Errorf("railway_custom_domain %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckTcpProxyDestroy verifies all railway_tcp_proxy resources have been deleted.
func testAccCheckTcpProxyDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_tcp_proxy" {
			continue
		}
		environmentId := rs.Primary.Attributes["environment_id"]
		serviceId := rs.Primary.Attributes["service_id"]
		response, err := getTcpProxy(ctx, client, environmentId, serviceId)
		if isNotFound(err) {
			continue // parent service is gone, so proxy is gone
		}
		if err != nil {
			return err
		}
		for _, proxy := range response.TcpProxies {
			if proxy.TCPProxy.Id == rs.Primary.ID {
				return fmt.Errorf("railway_tcp_proxy %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckWebhookDestroy verifies all railway_webhook resources have been deleted.
func testAccCheckWebhookDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_webhook" {
			continue
		}
		projectId := rs.Primary.Attributes["project_id"]
		response, err := getWebhooks(ctx, client, projectId)
		if isNotFound(err) {
			continue // parent project is gone, so webhook is gone
		}
		if err != nil {
			return err
		}
		for _, edge := range response.Webhooks.Edges {
			if edge.Node.Id == rs.Primary.ID {
				return fmt.Errorf("railway_webhook %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckDeploymentTriggerDestroy verifies all railway_deployment_trigger resources have been deleted.
func testAccCheckDeploymentTriggerDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_deployment_trigger" {
			continue
		}
		environmentId := rs.Primary.Attributes["environment_id"]
		projectId := rs.Primary.Attributes["project_id"]
		serviceId := rs.Primary.Attributes["service_id"]
		response, err := getDeploymentTriggers(ctx, client, environmentId, projectId, serviceId)
		if isNotFound(err) {
			continue // parent resource is gone, so trigger is gone
		}
		if err != nil {
			return err
		}
		for _, edge := range response.DeploymentTriggers.Edges {
			if edge.Node.DeploymentTrigger.Id == rs.Primary.ID {
				return fmt.Errorf("railway_deployment_trigger %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckEgressGatewayDestroy verifies all railway_egress_gateway resources have been deleted.
func testAccCheckEgressGatewayDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_egress_gateway" {
			continue
		}
		environmentId := rs.Primary.Attributes["environment_id"]
		serviceId := rs.Primary.Attributes["service_id"]
		response, err := getEgressGateways(ctx, client, environmentId, serviceId)
		if isNotFound(err) {
			continue // parent service is gone, so egress gateway is gone
		}
		if err != nil {
			return err
		}
		if len(response.EgressGateways) > 0 {
			return fmt.Errorf("railway_egress_gateway %s still exists after destroy", rs.Primary.ID)
		}
	}
	return nil
}

// testAccCheckPrivateNetworkDestroy verifies all railway_private_network resources have been deleted.
func testAccCheckPrivateNetworkDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_private_network" {
			continue
		}
		environmentId := rs.Primary.Attributes["environment_id"]
		response, err := getPrivateNetworks(ctx, client, environmentId)
		if isNotFound(err) {
			continue // parent environment is gone, so network is gone
		}
		if err != nil {
			return err
		}
		for _, network := range response.PrivateNetworks {
			if network.PrivateNetworkFields.PublicId == rs.Primary.ID {
				return fmt.Errorf("railway_private_network %s still exists after destroy", rs.Primary.ID)
			}
		}
	}
	return nil
}

// testAccCheckPrivateNetworkEndpointDestroy verifies all railway_private_network_endpoint resources have been deleted.
// Railway's getPrivateNetworkEndpoint returns stale data for 30+ seconds after deletion (no list endpoint exists),
// so we verify via the parent private network — if the network is gone, the endpoint is implicitly gone.
func testAccCheckPrivateNetworkEndpointDestroy(s *terraform.State) error {
	client := testAccNewClient()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "railway_private_network_endpoint" {
			continue
		}
		environmentId := rs.Primary.Attributes["environment_id"]
		// Verify via the parent private network being gone from the environment's network list
		err := testAccWaitUntilGone(func() error {
			response, err := getPrivateNetworks(ctx, client, environmentId)
			if isNotFound(err) {
				return &NotFoundError{ResourceType: "private network endpoint", Id: rs.Primary.ID}
			}
			if err != nil {
				return err
			}
			if len(response.PrivateNetworks) == 0 {
				return &NotFoundError{ResourceType: "private network endpoint", Id: rs.Primary.ID}
			}
			return nil // networks still exist, keep waiting
		})
		if err != nil {
			return fmt.Errorf("railway_private_network_endpoint %s: parent private network still exists after destroy", rs.Primary.ID)
		}
	}
	return nil
}
