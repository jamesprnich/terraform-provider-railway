---
page_title: "railway_trusted_domain Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway workspace trusted domain.
---

# railway_trusted_domain (Resource)

Railway workspace trusted domain. Used for SSO and access control. The domain must be verified via DNS before it is honoured.

~> **Workspace scope:** This resource affects workspace-wide SSO configuration. Changes here apply to **every project** in the workspace.

## Example Usage

```terraform
resource "railway_trusted_domain" "company" {
  workspace_id = var.railway_workspace_id
  domain_name  = "example.com"
  role         = "MEMBER"
}
```

## Schema

### Required

- `workspace_id` (String) Identifier of the workspace the trusted domain belongs to.
- `domain_name` (String) The fully-qualified domain name to trust (e.g. `example.com`).
- `role` (String) Role assigned to users authenticating from this domain (e.g. `MEMBER`).

### Read-Only

- `id` (String) Identifier of the trusted domain.
- `status` (String) Verification status of the trusted domain. One of `PENDING`, `VERIFIED`, `FAILED`.

## Import

Import is supported using the following syntax:

```shell
tofu import railway_trusted_domain.company <workspace_id>:<trusted_domain_id>
```
