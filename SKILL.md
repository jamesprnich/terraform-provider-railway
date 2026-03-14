---
name: railway-provider
description: Deploy and manage Railway infrastructure using the Railway Terraform Provider (v0.8.0)
---

# railway-provider

Use this skill when a project needs to deploy services to Railway using OpenTofu/Terraform with the `terraform-community-providers/railway` provider.

## Prerequisites

- Go 1.25+ (to build the provider from source)
- OpenTofu (or Terraform) installed
- `RAILWAY_TOKEN` environment variable set (team or project token)
- A `~/.terraformrc` dev override pointing at the built binary:
  ```hcl
  provider_installation {
    dev_overrides {
      "terraform-community-providers/railway" = "/path/to/go/bin"
    }
    direct {}
  }
  ```

## Actions

### Build the Provider

1. Clone the repo and build:
   ```bash
   go build -o ~/go/bin/terraform-provider-railway .
   ```
2. If any `.graphql` files were changed, regenerate the client first:
   ```bash
   go run github.com/Khan/genqlient
   ```
   Do NOT use `go generate` — that also requires a terraform binary.

### Deploy a Full Stack (Flask + Postgres Example)

The reference config is `examples/workflows/main.tf`. It creates a project, services, variables, service instances, and a public domain in a single apply.

1. Set variables:
   | Variable | Required | Default | Notes |
   |---|---|---|---|
   | `app_repo` | yes | — | GitHub `owner/repo` containing `examples/test-app/` |
   | `postgres_password` | yes | — | Postgres password (sensitive) |
   | `project_name` | no | `test-app` | Railway project name |

2. Apply:
   ```bash
   cd examples/workflows
   export RAILWAY_TOKEN="<token>"
   tofu apply \
     -var='app_repo=owner/repo' \
     -var='postgres_password=xxx' \
     -var='project_name=my-app'
   ```

3. Verify (wait 1-2 min for build/deploy). Use the `app_url` output:
   ```bash
   curl $(tofu output -raw app_url)/health   # → "ok"
   curl $(tofu output -raw app_url)/          # → Postgres version or graceful DB error
   ```

4. Idempotency check — a second `tofu plan` should show no changes.

5. Tear down:
   ```bash
   tofu destroy -var='app_repo=owner/repo' -var='postgres_password=xxx'
   ```

### Multi-Environment Architecture

For production setups with separate environments (dev, staging, production), use the pattern in `examples/workflows/environments/`. This creates a shared project and per-environment infrastructure using modules.

Key pattern:
- One `railway_project` with a default environment
- Additional `railway_environment` resources for staging/production
- Per-environment `railway_service_instance` resources for config (build commands, resource limits, healthcheck)
- Per-environment `railway_variable` resources for env-specific config
- Data sources (`data.railway_service`, `data.railway_environment`) to reference shared resources across environments

## Available Resources

| Resource | Purpose | Import Format |
|---|---|---|
| `railway_project` | Project with default environment | `<project_id>` |
| `railway_environment` | Additional environment within a project | `<project_id>:<name>` |
| `railway_service` | Service (image or GitHub repo), optional inline volume | `<service_id>` |
| `railway_service_instance` | Per-environment config: build/start commands, resource limits, healthcheck, region, replicas | `<service_id>:<environment_id>` |
| `railway_variable` | Environment variable scoped to service + environment | `<service_id>:<environment_name>:<name>` |
| `railway_variable_collection` | Bulk environment variables for a service + environment | `<service_id>:<environment_name>:<name1>:<name2>:...` |
| `railway_shared_variable` | Shared variable scoped to project + environment (no service) | `<project_id>:<environment_name>:<name>` |
| `railway_volume` | Standalone persistent volume attached to a service | `<project_id>:<volume_id>:<service_id>:<environment_id>` |
| `railway_volume_backup_schedule` | Automatic backup schedule for a volume instance | `<volume_instance_id>` |
| `railway_service_domain` | Auto-generated public `.up.railway.app` domain | `<service_id>:<environment_name>:<domain>` |
| `railway_custom_domain` | Custom domain with DNS verification | `<service_id>:<environment_name>:<domain>` |
| `railway_tcp_proxy` | TCP proxy for non-HTTP services | `<service_id>:<environment_id>:<tcp_proxy_id>` |
| `railway_webhook` | HTTP webhook notifications for project events | `<project_id>:<webhook_id>` |
| `railway_deployment_trigger` | Auto-deploy from GitHub/GitLab on push | `<project_id>:<environment_id>:<service_id>:<trigger_id>` |
| `railway_egress_gateway` | Static egress IP for external service allowlisting | `<service_id>:<environment_id>` |
| `railway_private_network` | Private network for internal service-to-service communication | `<environment_id>:<network_public_id>` |
| `railway_private_network_endpoint` | Connects a service to a private network with a DNS name | `<environment_id>:<private_network_id>:<service_id>` |

## Available Data Sources

| Data Source | Lookup By | Purpose |
|---|---|---|
| `data.railway_project` | `id` or `name` | Look up existing project |
| `data.railway_environment` | `id` or `project_id` + `name` | Look up existing environment |
| `data.railway_service` | `id` or `project_id` + `name` | Look up existing service |

## Patterns and Best Practices

### Service + Volume

For volumes, prefer the inline `volume` block on `railway_service` when the volume is tied to the default environment:

```terraform
resource "railway_service" "postgres" {
  name         = "postgres"
  project_id   = railway_project.main.id
  source_image = "postgres:17.5-alpine"

  volume = {
    name       = "pgdata"
    mount_path = "/data"
  }
}
```

Use standalone `railway_volume` only when you need a volume in a non-default environment.

### Service Instance Configuration

Use `railway_service_instance` for per-environment config. This is separate from `railway_service` because the same service can have different settings per environment:

```terraform
resource "railway_service_instance" "app" {
  service_id       = railway_service.app.id
  environment_id   = local.environment_id
  start_command    = "gunicorn app:app"
  build_command    = "pip install -r requirements.txt"
  healthcheck_path = "/health"
  num_replicas     = 2
  vcpus            = 1.0
  memory_gb        = 0.5
}
```

Note: `vcpus` and `memory_gb` are write-only — they can be set but not read back from the API. Import will not capture them.

### Private Networking

For inter-service communication (e.g., app → Postgres), use private networking instead of public URLs:

```terraform
resource "railway_private_network" "internal" {
  project_id     = railway_project.main.id
  environment_id = local.environment_id
  name           = "internal"
}

resource "railway_private_network_endpoint" "postgres" {
  private_network_id = railway_private_network.internal.id
  service_id         = railway_service.postgres.id
  environment_id     = local.environment_id
  service_name       = railway_service.postgres.name
}
```

Services can then reach each other via `<service-name>.railway.internal`.

### Deployment Triggers

Connect a GitHub repo to auto-deploy on push:

```terraform
resource "railway_deployment_trigger" "api" {
  service_id      = railway_service.api.id
  environment_id  = local.environment_id
  project_id      = railway_project.main.id
  repository      = "myorg/api"
  branch          = "main"
  source_provider = "github"
  check_suites    = true
}
```

### Egress Gateway (Static IP)

For services that need a fixed outbound IP (e.g., for external API allowlisting):

```terraform
resource "railway_egress_gateway" "api" {
  service_id     = railway_service.api.id
  environment_id = local.environment_id
}

# Use egress_gateway.api.ip_addresses to get the static IPs
```

### Volume Backups

Enable automatic backups for production volumes:

```terraform
resource "railway_volume_backup_schedule" "postgres_data" {
  volume_instance_id = railway_volume.postgres_data.id
  kinds              = ["DAILY", "WEEKLY"]
}
```

### Variables

Always set `PORT` as a variable on every service — Railway uses it for healthcheck probing and routing:

```terraform
resource "railway_variable" "app_port" {
  name           = "PORT"
  value          = "8080"
  environment_id = local.environment_id
  service_id     = railway_service.app.id
}
```

Use `railway_variable_collection` for bulk variables on a single service, `railway_shared_variable` for project-wide variables not scoped to a service.

### Webhooks

Send notifications when events occur in a project:

```terraform
resource "railway_webhook" "deploy_notifications" {
  project_id = railway_project.main.id
  url        = "https://example.com/webhooks/railway"
  filters    = ["deploy.completed"]
}
```

## Known Issues

### Private Networking Wireguard Bug

**What happens:** On the first Terraform-created deployment, `.railway.internal` DNS resolves correctly but TCP connections time out.

**Why:** Railway's Wireguard mesh doesn't establish the tunnel for Terraform-created services until the target service is redeployed. This is a Railway platform bug, not a provider issue.

**Workaround:** After first deploy, redeploy the target service from the Railway dashboard. Use `connect_timeout=5` in DATABASE_URL and handle DB errors gracefully so healthchecks still pass during the window before the tunnel is established.

### Service Domain Subdomain Not Customizable

**What happens:** `railway_service_domain` creates a domain with an auto-generated subdomain. You cannot choose the subdomain name.

**Why:** Railway's public GraphQL API does not support setting or renaming the subdomain. The `serviceDomainUpdate` mutation only supports changing `targetPort`.

**Workaround:** Use `railway_custom_domain` if you need a specific domain name.

### Write-Only Resource Limits

**What happens:** `vcpus` and `memory_gb` on `railway_service_instance` cannot be read back from the API after being set.

**Why:** Railway's `ServiceInstance` GraphQL type does not expose CPU/memory fields. They are set via a separate `serviceInstanceLimitsUpdate` mutation.

**Workaround:** The provider preserves these values from Terraform state/plan. Import will not capture them — you must declare them in config after import.

### Stale Data on Individual Resource Queries

**What happens:** After deleting a resource, querying it by ID (e.g., `environment(id: "...")`) can return the full resource data for 30+ seconds.

**Why:** Railway's API caches individual resource lookups. The list endpoint (e.g., `environments(projectId: "...")`) reflects deletions much faster (~1-2 seconds).

**Workaround:** The provider's Read methods use list queries filtered by ID instead of direct-by-ID queries where this is observed. Currently applied to `railway_environment`. If adding new resources, prefer list-based Read when the parent provides a list endpoint.

### "Operation Already in Progress" on Delete

**What happens:** Deleting a resource that is already being deleted returns `Cannot delete [resource]: an operation is already in progress`.

**Why:** Railway processes deletes asynchronously. A second delete request during processing is rejected. This is NOT matched by `isNotFound()`.

**Workaround:** All Delete methods treat "operation is already in progress" as successful (idempotent deletion). The disappears test helpers use `retry.RetryContext` polling to wait for deletion to complete.

## Guidelines

- NEVER use `go generate` — use `go run github.com/Khan/genqlient` to regenerate the GraphQL client
- GraphQL queries live in `internal/provider/*.graphql`, client code in `generated.go`
- Resources follow the pattern: model struct → Schema() → Configure() → CRUD methods → ImportState()
- Provider registration is in `provider.go` Resources() and DataSources() functions
- Registry docs in `docs/resources/` and `docs/data-sources/`
- Run `make test` to run all unit tests (mock-based, no Railway token needed)
- Run `make testacc` to run acceptance tests (requires `RAILWAY_TOKEN`)
- Both targets set the OpenTofu provider namespace env vars automatically
- If running tests manually (not via `make`), you must set these env vars for OpenTofu compatibility:
  ```bash
  TF_ACC_TERRAFORM_PATH="$(which tofu)" \
  TF_ACC_PROVIDER_NAMESPACE="hashicorp" \
  TF_ACC_PROVIDER_HOST="registry.opentofu.org" \
  go test ./internal/provider/ -v
  ```
- Run `./scripts/check-schema.sh` to verify the GraphQL schema hasn't drifted from the recorded version
