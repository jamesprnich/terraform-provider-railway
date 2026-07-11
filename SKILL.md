---
name: railway-provider
description: Deploy and manage Railway infrastructure using the Railway OpenTofu Provider (v0.11.1)
---

# railway-provider

Use this skill when a project needs to deploy services to Railway using OpenTofu with the `jamesprnich/railway` provider. Also compatible with Terraform.

## Prerequisites

- Go 1.25+ (to build the provider from source)
- OpenTofu (or Terraform) installed
- `RAILWAY_TOKEN` environment variable set (account token recommended, see [Authentication Guide](docs/guides/authentication.md))
- A `~/.tofurc` (or `~/.terraformrc`) dev override pointing at the built binary:
  ```hcl
  provider_installation {
    dev_overrides {
      "jamesprnich/railway" = "/path/to/go/bin"
    }
    direct {}
  }
  ```

## Actions

### Build the Provider

Clone the repo and build:
```bash
git clone https://github.com/jamesprnich/terraform-provider-railway.git
cd terraform-provider-railway
go build -o ~/go/bin/terraform-provider-railway .
```

### Deploy a Full Stack (Flask + Postgres Example)

The reference config is `examples/workflows/main.tf`. It creates a project, services, variable collections, service instances, and a public domain in a single apply.

1. Set variables:
   | Variable | Required | Default | Notes |
   |---|---|---|---|
   | `app_repo` | yes | — | GitHub `owner/repo` containing `examples/test-app/` |
   | `postgres_password` | yes | — | Postgres password (sensitive) |
   | `project_name` | no | `test-app` | Railway project name |

2. Init and apply:
   ```bash
   cd examples/workflows
   export RAILWAY_TOKEN="<token>"
   tofu init
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

### Deployment Pattern

Use a single `main.tf` that creates everything in one apply: project, services, variable collections, service instances, volumes, and domains. This is the tested and proven approach — see `examples/workflows/main.tf`.

For multiple environments, use separate `railway_environment` resources and scope `railway_service_instance`, `railway_variable_collection`, and `railway_volume` to each environment via `environment_id`.

**Minimise redeployments:** Railway triggers a redeployment every time a variable, service instance setting, or source connection changes. Use `railway_variable_collection` (not individual `railway_variable` resources) to set all variables for a service in one API call. Five individual variables = five queued redeployments. One collection = one redeployment.

**Partial failures:** If `tofu apply` fails partway through, run `tofu apply` again. The provider saves state after each successful resource creation, so retrying picks up where it left off — already-created resources are detected and skipped.

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
| `railway_deployment_trigger` | Auto-deploy from GitHub/GitLab on push | `<project_id>:<environment_id>:<service_id>:<trigger_id>` |
| `railway_egress_gateway` | Static egress IP for external service allowlisting | `<service_id>:<environment_id>` |
| `railway_private_network` | Private network for internal service-to-service communication | `<environment_id>:<network_public_id>` |
| `railway_private_network_endpoint` | Connects a service to a private network with a DNS name | `<environment_id>:<private_network_id>:<service_id>` |
| `railway_project_token` | Project-scoped deploy token for CI/CD (sensitive) | `<project_id>:<token_id>` |
| `railway_trusted_domain` | Workspace-level trusted domain for SSO | `<workspace_id>:<trusted_domain_id>` |
| `railway_notification_rule` | Notification rule (webhook, Slack, email) — replaces `railway_webhook` | `<workspace_id>:<rule_id>` |
| `railway_bucket` | S3-compatible object storage bucket (no delete API) | `<project_id>:<bucket_id>` |
| `railway_ssh_public_key` | SSH public key for workspace or authenticated user | `<key_id>` |
| `railway_project_member` | Project membership with role | `<project_id>:<user_id>` |

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

Use standalone `railway_volume` when you need a volume in a non-default environment, or when you need to reference the `volume_instance_id` (e.g., for `railway_volume_backup_schedule`). The inline volume block does not expose `volume_instance_id`.

### Service Instance Configuration

Use `railway_service_instance` for per-environment config. This is separate from `railway_service` because the same service can have different settings per environment:

```terraform
resource "railway_service_instance" "app" {
  service_id     = railway_service.app.id
  environment_id = local.environment_id

  # Source (optional — override what's set on railway_service)
  source_repo    = "myorg/myapp"
  root_directory = "/backend"
  config_path    = "backend/railway.toml"

  # Build & start
  build_command  = "pip install -r requirements.txt"
  start_command  = "gunicorn app:app"
  builder        = "RAILPACK"  # HEROKU, NIXPACKS, PAKETO, RAILPACK

  # Deploy settings
  region                     = "us-west1"
  num_replicas               = 2
  healthcheck_path           = "/health"
  healthcheck_timeout        = 300
  restart_policy_type        = "ON_FAILURE"  # ALWAYS, ON_FAILURE, NEVER
  restart_policy_max_retries = 3
  sleep_application          = false
  overlap_seconds            = 5
  draining_seconds           = 10
  watch_patterns             = ["backend/**"]
  pre_deploy_command         = ["python manage.py migrate"]
  # cron_schedule            = "0 */6 * * *"  # requires num_replicas = 1

  # Resource limits
  vcpus     = 1.0
  memory_gb = 0.5
}
```

Notes:
- `vcpus` and `memory_gb` are write-only — they can be set but not read back from the API. Import will not capture them.
- `region` uses Railway's `multiRegionConfig` internally. The `ServiceInstance.region` API field returns null — the provider reads region from `latestDeployment.meta`.
- `source_image` and `source_repo` are mutually exclusive. Use `source_image` for Docker images (e.g., `postgres:17`).

### Private Docker Registries (GHCR, Docker Hub private, ECR, etc.)

When `source_image` references a private image, supply `registry_credentials`. Requires a Railway **Pro** plan.

```terraform
resource "railway_service_instance" "backend" {
  service_id     = railway_service.backend.id
  environment_id = local.environment_id

  source_image = "ghcr.io/myorg/backend@sha256:abc123..."

  registry_credentials = {
    username = "myuser"               # GitHub username or token user
    password = var.ghcr_pull_token    # PAT with read:packages scope
  }
}

variable "ghcr_pull_token" {
  type      = string
  sensitive = true
}
```

Notes:
- `registry_credentials` requires `source_image` — it is a validation error without it.
- `password` is `Sensitive` (masked in plan output and state diffs) and write-only — Railway never returns it on read. The provider preserves it from state; no perpetual diff after apply.
- `registry_credentials` cannot be recovered on `tofu import` (same as `vcpus`/`memory_gb`). Re-set it in config after import.
- Public images (`source_image` without credentials) are unaffected — no change to existing behaviour.

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

Setting `source_repo` on `railway_service` automatically creates a deployment trigger — you do NOT need a separate `railway_deployment_trigger` for the same service. A second trigger on the same service will fail.

Use `railway_deployment_trigger` only when you need to manage the trigger separately from the service — for example, different branches per environment, `check_suites` gating, or monorepo `root_directory` filtering:

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

For services that need a fixed outbound IP (e.g., for external API allowlisting). Requires a **Pro plan** workspace. The service must have at least one deployment — the static IP is tied to the service's deployment region and activates after the next deploy. If `region` is omitted, Railway uses the service's current deployment region.

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
  volume_instance_id = railway_volume.postgres_data.volume_instance_id
  kinds              = ["DAILY", "WEEKLY"]
}
```

### Variables

Always set `PORT` as a variable on every service — Railway uses it for healthcheck probing and routing.

**Use `railway_variable_collection` for multiple variables on the same service.** Each individual `railway_variable` triggers a separate redeployment. A collection sets all variables in one API call — one redeployment instead of N:

```terraform
resource "railway_variable_collection" "app" {
  environment_id = local.environment_id
  service_id     = railway_service.app.id

  variables = [
    { name = "PORT", value = "8080" },
    { name = "DATABASE_URL", value = "postgresql://..." },
  ]
}
```

Use individual `railway_variable` only when you have a single variable that changes independently. Use `railway_shared_variable` for project-wide variables not scoped to a service.

### Notification Rules

Send notifications (via webhook, Slack, email, or other channels) when events occur. `railway_notification_rule` replaced `railway_webhook` in v0.9.0 to align with Railway's new notification model — webhooks are now one channel type among many.

```terraform
resource "railway_notification_rule" "deploy_notifications" {
  workspace_id = var.railway_workspace_id
  project_id   = railway_project.main.id
  event_types  = ["deployment.completed", "deployment.failed"]
  severities   = ["CRITICAL", "WARNING"]
  channel_configs = [
    jsonencode({
      type = "webhook"
      url  = "https://example.com/webhooks/railway"
    }),
  ]
}
```

`channel_configs` accepts a list of JSON strings — each describes one delivery channel. Refer to the Railway dashboard or API docs for the supported channel shapes.

## Known Issues

### Private Networking Wireguard Bug

**What happens:** On the first provider-created deployment, `.railway.internal` DNS resolves correctly but TCP connections time out.

**Why:** Railway's Wireguard mesh doesn't establish the tunnel for provider-created services until the target service is redeployed. This is a Railway platform bug, not a provider issue.

**Workaround:** After first deploy, redeploy the target service from the Railway dashboard. Use `connect_timeout=5` in DATABASE_URL and handle DB errors gracefully so healthchecks still pass during the window before the tunnel is established.

### Service Domain Subdomain Not Customizable

**What happens:** `railway_service_domain` creates a domain with an auto-generated subdomain. You cannot choose the subdomain name.

**Why:** Railway's public GraphQL API does not support setting or renaming the subdomain. The `serviceDomainUpdate` mutation only supports changing `targetPort`.

**Workaround:** Use `railway_custom_domain` if you need a specific domain name.

### Write-Only Resource Limits

**What happens:** `vcpus` and `memory_gb` on `railway_service_instance` cannot be read back from the API after being set.

**Why:** Railway's `ServiceInstance` GraphQL type does not expose CPU/memory fields. They are set via a separate `serviceInstanceLimitsUpdate` mutation.

**Workaround:** The provider preserves these values from state/plan. Import will not capture them — you must declare them in config after import.

### Multiple Redeployments on First Apply

**What happens:** On initial `tofu apply`, each service shows 3-4 queued deployments in the Railway dashboard instead of one.

**Why:** Railway triggers a redeployment on every mutation — service source connection, variable changes, and service instance config updates are separate API calls. The Railway UI avoids this by batching changes locally and deploying once, but the API has no equivalent "hold deployments" mechanism.

**Workaround:** Use `railway_variable_collection` instead of individual `railway_variable` resources to minimise redeployments (one collection = one redeployment instead of one per variable). Beyond that, 3-4 deployments per service on first apply is the practical minimum with the current API. Subsequent `tofu apply` with no changes triggers zero redeployments. This is a Railway API limitation, not a provider bug.

