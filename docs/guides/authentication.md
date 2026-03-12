---
page_title: "Authentication Guide"
---

# Authentication Guide

The Railway Terraform provider supports two types of API tokens. Choosing the right token type is critical for both security and functionality — they have very different permission levels.

## Token Types

### Account Tokens (Recommended for Terraform)

Account tokens have full access to all projects and resources within your Railway account.

**How to create:** Railway Dashboard > Account Settings > Tokens > Create Token

**Permissions:**
- Create, read, update, and delete services
- Connect Docker images and GitHub repos to services (`serviceConnect`)
- Manage volumes, domains, variables, and all other resources
- Access all projects in the account

**When to use:** Any Terraform workflow that needs to create or fully configure services. This is the token type you need for most Terraform operations.

**Security note:** Account tokens are powerful. Never commit them to source control. Use environment variables or a secrets manager:

```bash
export RAILWAY_TOKEN="your-account-token"
```

### Project Tokens

Project tokens are scoped to a single environment within a single project. They provide strong isolation but have limited permissions.

**How to create:** Railway Dashboard > Project > Settings > Tokens > Create Token

**Permissions:**
- Read project, environment, and service metadata
- Create and delete services
- Manage deployment triggers, egress gateways, private networks
- Delete resources within the scoped environment

**Limitations (will cause Terraform errors):**
- **Cannot run `serviceConnect`** — this means you cannot attach a Docker image or GitHub repo to a service. Any `railway_service` resource with `source_image` or `source_repo` set will partially create (the service is created but the source is not attached) and then error.
- **Cannot list all projects** — the `data.railway_project` data source with a `name` lookup will fail. Direct `id` lookup works.
- **Cannot query across environments** — only the environment the token is scoped to is accessible.

**Authentication header:** Project tokens use a different HTTP header than account tokens. The provider handles this automatically by sending both headers:
- `Authorization: Bearer <token>` (used by account tokens)
- `Project-Access-Token: <token>` (used by project tokens)

**When to use:** Read-only operations, managing resources on existing services, or CI/CD pipelines that only need to trigger deployments within a specific environment.

## Token Type Comparison

| Capability | Account Token | Project Token |
|-----------|:---:|:---:|
| Create services | Yes | Yes |
| Attach Docker image / repo (`serviceConnect`) | Yes | **No** |
| Create volumes | Yes | Yes |
| Manage domains | Yes | Yes |
| Manage deployment triggers | Yes | Yes |
| Manage egress gateways | Yes | Yes |
| Manage private networks | Yes | Yes |
| Manage variables | Yes | Yes |
| List all projects | Yes | **No** |
| Access multiple environments | Yes | **No** |
| Access multiple projects | Yes | **No** |

## Recommended Setup

For a typical Terraform workflow managing Railway infrastructure:

1. **Create a dedicated account token** for Terraform automation
2. **Store it in your CI/CD secrets** or a secrets manager
3. **Scope your Terraform configs** to specific projects using `project_id` — the isolation comes from your config, not the token
4. **Rotate tokens periodically** via the Railway dashboard

```terraform
provider "railway" {
  # Set via RAILWAY_TOKEN environment variable
  # Do not hardcode tokens in Terraform files
}

resource "railway_service" "web" {
  name         = "my-app"
  project_id   = "your-project-id"
  source_image = "nginx:1.27.5-alpine"
}
```

## Troubleshooting

### "Not Authorized" errors

If you see `Not Authorized` errors, check:

1. **Token type** — Are you using a project token for an operation that requires an account token? See the comparison table above.
2. **Environment scope** — Project tokens only work within their scoped environment. Passing a different `environment_id` will fail.
3. **Token validity** — Tokens can be revoked from the Railway dashboard. Generate a new one if in doubt.

### "serviceConnect Not Authorized"

This specifically means you're using a project token and trying to set `source_image` or `source_repo` on a service. Switch to an account token.
