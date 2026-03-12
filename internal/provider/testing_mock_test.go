package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// graphqlRequest is the structure of a GraphQL request body.
type graphqlRequest struct {
	OperationName string          `json:"operationName"`
	Query         string          `json:"query"`
	Variables     json.RawMessage `json:"variables"`
}

// mockFixtures maps GraphQL operation names to JSON response bodies.
type mockFixtures map[string]string

// newMockGraphQLServer creates an httptest.Server that returns canned responses
// based on the GraphQL operation name. The fixtures map operation names to JSON
// response strings. Unknown operations cause a test failure.
func newMockGraphQLServer(t *testing.T, fixtures mockFixtures) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("mock server: failed to decode request body: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response, ok := fixtures[req.OperationName]
		if !ok {
			t.Errorf("mock server: unexpected operation %q", req.OperationName)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"message":"unexpected operation: %s"}]}`, req.OperationName)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, response)
	}))
}

// testUnitProtoV6ProviderFactories returns provider factories for use with
// resource.UnitTest. These do not require RAILWAY_TOKEN or TF_ACC.
func testUnitProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"railway": providerserver.NewProtocol6WithError(New("test")()),
	}
}

// testUnitProviderConfig returns an HCL provider block pointing at the mock server.
func testUnitProviderConfig(serverURL string) string {
	return fmt.Sprintf(`
provider "railway" {
  token   = "test-token"
  api_url = "%s"
}
`, serverURL)
}
