package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"railway": providerserver.NewProtocol6WithError(New("test")()),
}

// Shared fixture IDs — populated by TestMain for acceptance tests.
var (
	testAccWorkspaceId     = os.Getenv("RAILWAY_TEST_WORKSPACE_ID")
	testAccProjectId       string
	testAccServiceId       string
	testAccEnvironmentId   string
	testAccEnvironmentName string
	testAccProjectName     string
	testAccServiceName     = "acc-test-service"
)

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("RAILWAY_TOKEN"); v == "" {
		t.Fatal("RAILWAY_TOKEN must be set for acceptance tests")
	}
}

// testFixtureFile is the path where we persist fixture IDs between test runs.
// If the test process crashes, the next run reads this file and cleans up
// the orphaned project before creating a new one.
var testFixtureFile = filepath.Join(os.TempDir(), "railway-tf-acc-fixtures.json")

type testFixtures struct {
	ProjectId string `json:"project_id"`
}

func writeFixtureFile(projectId string) {
	data, _ := json.Marshal(testFixtures{ProjectId: projectId})
	if err := os.WriteFile(testFixtureFile, data, 0o600); err != nil {
		log.Printf("writeFixtureFile: WARN — failed to persist %s: %s", testFixtureFile, err)
	}
}

func removeFixtureFile() {
	if err := os.Remove(testFixtureFile); err != nil && !os.IsNotExist(err) {
		log.Printf("removeFixtureFile: WARN — failed to remove %s: %s", testFixtureFile, err)
	}
}

// cleanupOrphanedFixtures reads the fixture file from a previous run and
// deletes the orphaned project if it still exists.
func cleanupOrphanedFixtures(ctx context.Context, client graphql.Client) {
	// The fixture-file path is a fixed test-time value under the OS temp dir,
	// not user input; the gosec G304 warning does not apply.
	data, err := os.ReadFile(testFixtureFile) //nolint:gosec // fixed test path
	if err != nil {
		return // no previous fixture file
	}

	var prev testFixtures
	if json.Unmarshal(data, &prev) != nil || prev.ProjectId == "" {
		removeFixtureFile()
		return
	}

	log.Printf("TestMain: found orphaned fixture file — cleaning up project %s", prev.ProjectId)
	_, err = deleteProject(ctx, client, prev.ProjectId)
	if err != nil && !isNotFound(err) {
		log.Printf("TestMain: WARNING — failed to delete orphaned project %s: %s", prev.ProjectId, err)
	} else {
		log.Printf("TestMain: cleaned up orphaned project %s", prev.ProjectId)
	}

	removeFixtureFile()
}

// TestMain manages shared test fixtures for acceptance tests.
// When TF_ACC is set, it creates a project + service before any tests run,
// and deletes the project (cascading all children) after all tests complete.
// If the previous run crashed, orphaned fixtures are cleaned up first via
// the persisted fixture file.
func TestMain(m *testing.M) {
	// For unit tests (no TF_ACC), skip fixture setup entirely.
	if os.Getenv("TF_ACC") == "" {
		os.Exit(m.Run())
	}

	if testAccWorkspaceId == "" {
		log.Fatal("RAILWAY_TEST_WORKSPACE_ID must be set for acceptance tests")
	}

	ctx := context.Background()
	client := testAccNewClient()

	// Clean up any orphaned fixtures from a previous crashed run.
	cleanupOrphanedFixtures(ctx, client)

	// Create test project with a unique name.
	testAccProjectName = fmt.Sprintf("tf-acc-%d", time.Now().Unix())
	projectResp, err := createProject(ctx, client, ProjectCreateInput{
		Name:                   testAccProjectName,
		DefaultEnvironmentName: "production",
		WorkspaceId:            &testAccWorkspaceId,
	})
	if err != nil {
		log.Fatalf("TestMain: failed to create test project: %s", err)
	}

	project := projectResp.ProjectCreate.Project
	testAccProjectId = project.Id

	// Persist fixture ID immediately so a crash leaves a breadcrumb for the next run.
	writeFixtureFile(testAccProjectId)

	if len(project.Environments.Edges) != 1 {
		_, _ = deleteProject(ctx, client, testAccProjectId)
		log.Fatalf("TestMain: expected 1 environment, got %d", len(project.Environments.Edges))
	}

	testAccEnvironmentId = project.Environments.Edges[0].Node.Id
	testAccEnvironmentName = project.Environments.Edges[0].Node.Name

	// Create test service with a source image to trigger a deployment.
	// Egress gateway requires at least one deployment on the service.
	serviceName := testAccServiceName
	serviceResp, err := createService(ctx, client, ServiceCreateInput{
		Name:      &serviceName,
		ProjectId: testAccProjectId,
	})
	if err != nil {
		_, _ = deleteProject(ctx, client, testAccProjectId)
		log.Fatalf("TestMain: failed to create test service: %s", err)
	}

	testAccServiceId = serviceResp.ServiceCreate.Id

	// Attach an image to the service's instance in the default env. Uses the
	// env-scoped serviceInstanceUpdate — the env-less serviceConnect mutation
	// was removed in v0.11.0 because it would create source connections
	// across every non-fork environment.
	image := "nginx:1.27-alpine"
	_, err = updateServiceInstanceInEnvironment(ctx, client, testAccEnvironmentId, testAccServiceId, ServiceInstanceUpdateInput{
		Source: &ServiceSourceInput{Image: &image},
	})
	if err != nil {
		_, _ = deleteProject(ctx, client, testAccProjectId)
		log.Fatalf("TestMain: failed to attach image to service instance: %s", err)
	}

	// Wait for the service instance to exist (deployment initiated).
	err = retry.RetryContext(ctx, 60*time.Second, func() *retry.RetryError {
		_, err := getServiceInstance(ctx, client, testAccEnvironmentId, testAccServiceId)
		if err != nil {
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		_, _ = deleteProject(ctx, client, testAccProjectId)
		log.Fatalf("TestMain: timed out waiting for service instance: %s", err)
	}

	log.Printf("TestMain: fixtures ready — project=%s env=%s service=%s",
		testAccProjectId, testAccEnvironmentId, testAccServiceId)

	// Run all tests.
	code := m.Run()

	// Cleanup: delete project (cascading all child resources).
	// This is the safety net — individual tests verify their own cleanup via CheckDestroy.
	log.Printf("TestMain: cleaning up project %s (%s)", testAccProjectName, testAccProjectId)
	_, cleanupErr := deleteProject(ctx, client, testAccProjectId)
	if cleanupErr != nil {
		log.Printf("TestMain: WARNING — failed to delete project %s: %s", testAccProjectId, cleanupErr)
	} else {
		removeFixtureFile()
	}

	os.Exit(code)
}
