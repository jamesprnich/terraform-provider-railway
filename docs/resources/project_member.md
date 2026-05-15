---
page_title: "railway_project_member Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway project membership.
---

# railway_project_member (Resource)

Railway project membership. Adds a user to a project with a given role.

~> **Invitation flow not handled here:** This resource calls `projectMemberAdd`, which requires the user to already exist in Railway. To invite a user by email (sending them a join link), use the Railway dashboard's invitation flow or call `projectInvitationCreate` directly — those are outside the scope of this Terraform resource.

## Example Usage

```terraform
resource "railway_project_member" "alice" {
  project_id = railway_project.main.id
  user_id    = "user-12345"
  role       = "MEMBER"
}
```

## Schema

### Required

- `project_id` (String) Identifier of the project.
- `user_id` (String) Identifier of the user to add as a member.
- `role` (String) Role assigned to the member. One of `ADMIN`, `MEMBER`, `VIEWER`.

### Read-Only

- `id` (String) Identifier of the project membership.
- `email` (String) Email address of the member.
- `name` (String) Display name of the member.

## Import

Import is supported using the following syntax:

```shell
tofu import railway_project_member.alice <project_id>:<user_id>
```
