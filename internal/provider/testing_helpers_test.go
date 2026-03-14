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
