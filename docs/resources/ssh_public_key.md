---
page_title: "railway_ssh_public_key Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway SSH public key.
---

# railway_ssh_public_key (Resource)

Railway SSH public key. Registers an SSH key against a workspace (or the authenticated user if `workspace_id` is omitted) for use with Railway features that require SSH authentication.

~> **Workspace scope:** When `workspace_id` is set, this resource affects workspace-level SSH key configuration that applies to **every member** of that workspace.

## Example Usage

```terraform
resource "railway_ssh_public_key" "ci" {
  name         = "github-actions"
  public_key   = file("~/.ssh/id_ed25519.pub")
  workspace_id = var.railway_workspace_id
}
```

## Schema

### Required

- `name` (String) Friendly name for the SSH key.
- `public_key` (String) OpenSSH-format public key (e.g. `ssh-ed25519 AAAA... user@host`).

### Optional

- `workspace_id` (String) Identifier of the workspace the key belongs to. If omitted, the key is registered against the authenticated user.

### Read-Only

- `id` (String) Identifier of the SSH key.
- `fingerprint` (String) Server-computed fingerprint of the public key.

## Import

Import is supported using the following syntax:

```shell
tofu import railway_ssh_public_key.ci <key_id>
```
