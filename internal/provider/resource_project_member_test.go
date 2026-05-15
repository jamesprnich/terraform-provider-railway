package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance test is intentionally skipped — project member operations require a real
// userId (a user that already exists in Railway), and adding/removing members affects
// real users' access. Requires explicit user approval, a dedicated test user, and manual
// validation in an isolated workspace to enable.
func TestAccProjectMemberResourceDefault(t *testing.T) {
	t.Skip("requires a real Railway userId — adding/removing real users to projects has side-effects beyond fixture cleanup. Test manually with a dedicated test user.")
}

func TestProjectMemberResource_basic(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"
	userId := "user-abc-123"

	server := newMockGraphQLServer(t, mockFixtures{
		"addProjectMember":    `{"data":{"projectMemberAdd":{"id":"` + userId + `","email":"user@example.com","name":"Test User","role":"MEMBER"}}}`,
		"getProjectMembers":   `{"data":{"projectMembers":[{"id":"` + userId + `","email":"user@example.com","name":"Test User","role":"MEMBER"}]}}`,
		"removeProjectMember": `{"data":{"projectMemberRemove":[]}}`,
	})
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(server.URL) + `
resource "railway_project_member" "test" {
  project_id = "` + projectId + `"
  user_id    = "` + userId + `"
  role       = "MEMBER"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_project_member.test", "id", userId),
					resource.TestCheckResourceAttr("railway_project_member.test", "project_id", projectId),
					resource.TestCheckResourceAttr("railway_project_member.test", "user_id", userId),
					resource.TestCheckResourceAttr("railway_project_member.test", "role", "MEMBER"),
					resource.TestCheckResourceAttr("railway_project_member.test", "email", "user@example.com"),
					resource.TestCheckResourceAttr("railway_project_member.test", "name", "Test User"),
				),
			},
		},
	})
}

func TestProjectMemberResource_disappears(t *testing.T) {
	projectId := "11111111-2222-3333-4444-555555555555"
	userId := "user-def-456"

	srv, disappear := newDisappearsMockServer(t, mockFixtures{
		"addProjectMember":    `{"data":{"projectMemberAdd":{"id":"` + userId + `","email":"user@example.com","name":"Test User","role":"MEMBER"}}}`,
		"getProjectMembers":   `{"data":{"projectMembers":[{"id":"` + userId + `","email":"user@example.com","name":"Test User","role":"MEMBER"}]}}`,
		"removeProjectMember": `{"data":{"projectMemberRemove":[]}}`,
	}, "getProjectMembers")
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testUnitProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUnitProviderConfig(srv.URL) + `
resource "railway_project_member" "test" {
  project_id = "` + projectId + `"
  user_id    = "` + userId + `"
  role       = "MEMBER"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_project_member.test", "id", userId),
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
