package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"railway": providerserver.NewProtocol6WithError(New("test")()),
}

// Acceptance test environment IDs.
// Override via env vars for different Railway accounts/projects.
var (
	testAccProjectId     = envOrDefault("RAILWAY_TEST_PROJECT_ID", "ac226181-814e-4e7e-a89c-76f8a44cd924")
	testAccWorkspaceId   = envOrDefault("RAILWAY_TEST_WORKSPACE_ID", "1ea62ece-49ff-4106-808a-cd652d6c87b1")
	testAccServiceId     = envOrDefault("RAILWAY_TEST_SERVICE_ID", "043a0223-69b8-42cf-89c9-d38b5138f846")
	testAccEnvironmentId = envOrDefault("RAILWAY_TEST_ENVIRONMENT_ID", "afc4a8f4-5f4c-4b9b-921f-213f013984f9")
	testAccDefaultRegion  = envOrDefault("RAILWAY_TEST_DEFAULT_REGION", "us-west2")
	testAccEnvironmentName = envOrDefault("RAILWAY_TEST_ENVIRONMENT_NAME", "production")
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("RAILWAY_TOKEN"); v == "" {
		t.Fatal("RAILWAY_TOKEN must be set for acceptance tests")
	}
}
