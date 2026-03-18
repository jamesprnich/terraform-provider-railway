# Railway OpenTofu Provider

An [OpenTofu](https://opentofu.org) provider for managing [Railway](https://railway.app) infrastructure as code. Also compatible with [Terraform](https://www.terraform.io).

17 resources, 3 data sources, full import support.

## Quick Start

```hcl
terraform {
  required_providers {
    railway = {
      source  = "jamesprnich/railway"
      version = "~> 0.8.0"
    }
  }
}

provider "railway" {
  # Set RAILWAY_TOKEN environment variable
}

resource "railway_project" "main" {
  name = "my-app"
}

resource "railway_service" "api" {
  name         = "api"
  project_id   = railway_project.main.id
  source_image = "node:22-alpine"
}
```

## Documentation

Full documentation with guides, resource references, and examples:

**[jamesprnich.github.io/terraform-provider-railway](https://jamesprnich.github.io/terraform-provider-railway/)**

## Resources

| Resource | Purpose |
|---|---|
| `railway_project` | Project with default environment |
| `railway_environment` | Additional environment within a project |
| `railway_service` | Service (Docker image or GitHub repo) with optional inline volume |
| `railway_service_instance` | Per-environment config: source, build, deploy settings, resource limits |
| `railway_variable` | Environment variable scoped to service + environment |
| `railway_variable_collection` | Bulk environment variables for a service + environment |
| `railway_shared_variable` | Project-wide variable (not scoped to a service) |
| `railway_volume` | Standalone persistent volume |
| `railway_volume_backup_schedule` | Automatic backup schedule for a volume instance |
| `railway_service_domain` | Auto-generated `.up.railway.app` domain |
| `railway_custom_domain` | Custom domain with DNS verification |
| `railway_tcp_proxy` | TCP proxy for non-HTTP services |
| `railway_webhook` | HTTP webhook notifications for project events |
| `railway_deployment_trigger` | Auto-deploy from GitHub/GitLab on push |
| `railway_egress_gateway` | Static egress IP for external service allowlisting |
| `railway_private_network` | Private network for internal service-to-service communication |
| `railway_private_network_endpoint` | Connects a service to a private network with DNS name |

## Data Sources

| Data Source | Lookup By |
|---|---|
| `data.railway_project` | `id` or `name` |
| `data.railway_environment` | `id` or `project_id` + `name` |
| `data.railway_service` | `id` or `project_id` + `name` |

## AI Agent Integration

This provider ships with [`SKILL.md`](SKILL.md) — a structured reference designed for AI coding agents. It contains every resource, every field, every import format, working examples, and known issues. Point your AI agent at this file and it can deploy full Railway infrastructure autonomously.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, testing, and development workflow.

## License

MPL-2.0
