package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance test is intentionally skipped — SSH public keys are WORKSPACE- or USER-LEVEL
// resources. Creating one against the live API would persist outside the fixture project
// (in the workspace's or authenticated user's key list). Requires explicit user approval
// and manual validation in an isolated workspace to enable.
func TestAccSshPublicKeyResourceDefault(t *testing.T) {
	t.Skip("workspace- or user-level resource — would persist outside the fixture project. Test manually in an isolated workspace.")
}

func TestSshPublicKeyResource_basic(t *testing.T) {
	server := newMockGraphQLServer(t, mockFixtures{
		"createSshPublicKey": `{"data":{"sshPublicKeyCreate":{"id":"key-1","name":"ci","publicKey":"ssh-ed25519 AAAA fake","fingerprint":"SHA256:abcdef","workspaceId":"ws-1","userId":null}}}`,
		"getSshPublicKeys":   `{"data":{"sshPublicKeys":{"edges":[{"node":{"id":"key-1","name":"ci","publicKey":"ssh-ed25519 AAAA fake","fingerprint":"SHA256:abcdef","workspaceId":"ws-1","userId":null}}]}}}`,
		"deleteSshPublicKey": `{"data":{"sshPublicKeyDelete":true}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_ssh_public_key" "test" {
  name         = "ci"
  public_key   = "ssh-ed25519 AAAA fake"
  workspace_id = "ws-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_ssh_public_key.test", "id", "key-1"),
					resource.TestCheckResourceAttr("railway_ssh_public_key.test", "name", "ci"),
					resource.TestCheckResourceAttr("railway_ssh_public_key.test", "fingerprint", "SHA256:abcdef"),
					resource.TestCheckResourceAttr("railway_ssh_public_key.test", "workspace_id", "ws-1"),
				),
			},
		},
	})
}

func TestSshPublicKeyResource_disappears(t *testing.T) {
	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"createSshPublicKey": `{"data":{"sshPublicKeyCreate":{"id":"key-2","name":"ci","publicKey":"ssh-ed25519 AAAA fake","fingerprint":"SHA256:abcdef","workspaceId":"ws-1","userId":null}}}`,
		"getSshPublicKeys":   `{"data":{"sshPublicKeys":{"edges":[{"node":{"id":"key-2","name":"ci","publicKey":"ssh-ed25519 AAAA fake","fingerprint":"SHA256:abcdef","workspaceId":"ws-1","userId":null}}]}}}`,
		"deleteSshPublicKey": `{"data":{"sshPublicKeyDelete":true}}`,
	}, "getSshPublicKeys")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_ssh_public_key" "test" {
  name         = "ci"
  public_key   = "ssh-ed25519 AAAA fake"
  workspace_id = "ws-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_ssh_public_key.test", "id", "key-2"),
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
