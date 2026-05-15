---
page_title: "railway_project_token Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway project token for CI/CD authentication.
---

# railway_project_token (Resource)

Railway project token. Generates a scoped deploy token for CI/CD pipelines.

~> **Note:** The `token` attribute is only available at creation time. Railway does not return the raw token on subsequent reads or imports. Store it in your CI secret manager immediately after `tofu apply`.

## Example Usage

```terraform
resource "railway_project_token" "ci" {
  name           = "github-actions"
  project_id     = railway_project.example.id
  environment_id = railway_project.example.default_environment.id
}

output "deploy_token" {
  value     = railway_project_token.ci.token
  sensitive = true
}
```

## Schema

### Required

- `name` (String) Name of the project token.
- `project_id` (String) Identifier of the project the token belongs to.
- `environment_id` (String) Identifier of the environment the token has access to.

### Read-Only

- `id` (String) Identifier of the project token.
- `token` (String, Sensitive) The raw token value. Only populated at creation; null on import or read.

## Import

Import is supported using the following syntax:

```shell
tofu import railway_project_token.ci <project_id>:<token_id>
```

Note: the raw token value is **not** recoverable on import. Regenerate the token if you need the value.
