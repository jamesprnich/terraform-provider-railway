---
title: Getting Started
description: Railway OpenTofu provider — installation, authentication, and quick start guide.
---

# Railway OpenTofu Provider

An [OpenTofu](https://opentofu.org) provider for managing [Railway](https://railway.app) infrastructure as code. Also compatible with [Terraform](https://www.terraform.io).

17 resources, 3 data sources, full import support.

## Installation

Add the provider to your `required_providers` block:

```hcl
terraform {
  required_providers {
    railway = {
      source  = "jamesprnich/railway"
      version = "~> 0.8.0"
    }
  }
}
```

## Authentication

Set the `RAILWAY_TOKEN` environment variable:

```bash
export RAILWAY_TOKEN="your-token"
```

Or configure it in the provider block:

```hcl
provider "railway" {
  token = var.railway_token
}
```

!!! tip "Token types"
    Railway has two token types with very different permissions. **Account tokens** are recommended — they have full access to all resources. **Project tokens** are scoped to a single environment and cannot attach sources to services. See the [Authentication Guide](guides/authentication.md) for details.

## Quick Start

```hcl
resource "railway_project" "main" {
  name = "my-app"
}

resource "railway_service" "postgres" {
  name         = "postgres"
  project_id   = railway_project.main.id
  source_image = "postgres:17.5-alpine"

  volume = {
    name       = "pgdata"
    mount_path = "/var/lib/postgresql/data"
  }
}

resource "railway_variable" "postgres_password" {
  name           = "POSTGRES_PASSWORD"
  value          = var.postgres_password
  environment_id = railway_project.main.default_environment.id
  service_id     = railway_service.postgres.id
}
```

```bash
tofu apply -var='postgres_password=secretpassword'
```

## Resources

| Resource | Purpose |
|---|---|
| [`railway_project`](resources/project.md) | Project with default environment |
| [`railway_environment`](resources/environment.md) | Additional environment within a project |
| [`railway_service`](resources/service.md) | Service (Docker image or GitHub repo) with optional inline volume |
| [`railway_service_instance`](resources/service_instance.md) | Per-environment config: source, build, deploy settings, resource limits |
| [`railway_variable`](resources/variable.md) | Environment variable scoped to service + environment |
| [`railway_variable_collection`](resources/variable_collection.md) | Bulk environment variables for a service + environment |
| [`railway_shared_variable`](resources/shared_variable.md) | Project-wide variable (not scoped to a service) |
| [`railway_volume`](resources/volume.md) | Standalone persistent volume |
| [`railway_volume_backup_schedule`](resources/volume_backup_schedule.md) | Automatic backup schedule for a volume instance |
| [`railway_service_domain`](resources/service_domain.md) | Auto-generated `.up.railway.app` domain |
| [`railway_custom_domain`](resources/custom_domain.md) | Custom domain with DNS verification |
| [`railway_tcp_proxy`](resources/tcp_proxy.md) | TCP proxy for non-HTTP services |
| [`railway_webhook`](resources/webhook.md) | HTTP webhook notifications for project events |
| [`railway_deployment_trigger`](resources/deployment_trigger.md) | Auto-deploy from GitHub/GitLab on push |
| [`railway_egress_gateway`](resources/egress_gateway.md) | Static egress IP for external service allowlisting |
| [`railway_private_network`](resources/private_network.md) | Private network for internal service-to-service communication |
| [`railway_private_network_endpoint`](resources/private_network_endpoint.md) | Connects a service to a private network with DNS name |

## Data Sources

| Data Source | Lookup By |
|---|---|
| [`data.railway_project`](data-sources/project.md) | `id` or `name` |
| [`data.railway_environment`](data-sources/environment.md) | `id` or `project_id` + `name` |
| [`data.railway_service`](data-sources/service.md) | `id` or `project_id` + `name` |

## Known Issues

- **Webhook types not in public schema** — `railway_webhook` works with mock tests but live API calls will fail until Railway re-adds webhook types to their public GraphQL schema.
- **Private networking** requires a manual redeploy of the target service after the first provider-created deployment (Railway platform bug with Wireguard tunnel setup).
- **Service domain subdomains** are auto-generated and cannot be customized via the API. Use `railway_custom_domain` for specific domain names.
- **`vcpus` and `memory_gb`** on `railway_service_instance` are write-only — they can be set but not read back. Import will not capture them.

## AI Agent Integration

This provider ships with [`SKILL.md`](https://github.com/jamesprnich/terraform-provider-railway/blob/main/SKILL.md) — a structured reference designed for AI coding agents. It contains every resource, every field, every import format, working examples, and known issues. Point your AI agent at this file and it can deploy full Railway infrastructure autonomously.
