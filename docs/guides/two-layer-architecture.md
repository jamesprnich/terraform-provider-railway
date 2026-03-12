---
page_title: "Two-Layer Architecture Guide"
---

# Two-Layer Architecture Guide

Managing Railway infrastructure with Terraform/OpenTofu works best when split into two layers: **services** and **environments**. This guide explains why, and how to structure your configuration.

## The Problem

In Railway, a **Service** is a project-level resource that exists across all environments. When you attach a source (Docker image or GitHub repo) to a service, Railway immediately deploys it into the default environment (production).

This creates two problems for infrastructure-as-code:

1. **Unwanted deployments** — Creating a service with `source_image = "postgres:17"` triggers an immediate production deployment, even if you're not ready.
2. **No per-environment versioning** — The source is set at the service level. You can't run `postgres:17` in production and test `postgres:18` in dev if the image is set on the service itself.

## The Solution: Two Layers

### Layer 1 — Services (run once)

Create the Railway project and empty services. No source images, no source repos, no volumes. Just the skeleton.

```terraform
resource "railway_project" "main" {
  name = "my-app"
}

resource "railway_service" "backend" {
  name       = "backend"
  project_id = railway_project.main.id
}

resource "railway_service" "frontend" {
  name       = "frontend"
  project_id = railway_project.main.id
}

resource "railway_service" "postgres" {
  name       = "postgres"
  project_id = railway_project.main.id
}
```

This creates three empty services. Nothing deploys. Nothing runs.

### Layer 2 — Environments (run per environment)

Use `railway_service_instance` to configure each service in each environment. This is where you set the source, build settings, resource limits, and everything else that varies per environment.

```terraform
data "railway_project" "main" {
  name = "my-app"
}

data "railway_service" "backend" {
  project_id = data.railway_project.main.id
  name       = "backend"
}

data "railway_service" "postgres" {
  project_id = data.railway_project.main.id
  name       = "postgres"
}

resource "railway_environment" "this" {
  name       = "dev"
  project_id = data.railway_project.main.id
}

resource "railway_service_instance" "backend" {
  service_id     = data.railway_service.backend.id
  environment_id = railway_environment.this.id
  source_repo    = "myorg/my-app"
  root_directory = "backend"
  vcpus          = 0.5
  memory_gb      = 0.5
}

resource "railway_service_instance" "postgres" {
  service_id     = data.railway_service.postgres.id
  environment_id = railway_environment.this.id
  source_image   = "postgres:17.5-alpine"
}

resource "railway_volume" "pgdata" {
  name           = "pgdata"
  project_id     = data.railway_project.main.id
  service_id     = data.railway_service.postgres.id
  environment_id = railway_environment.this.id
  mount_path     = "/var/lib/postgresql/data"
}
```

Now you control exactly what runs in each environment. Dev can test `postgres:18` while production stays on `postgres:17.5`. Backend resource limits can be smaller in dev and larger in production.

## Data Sources Replace Shared Variables

The environment layer discovers services by name using data sources. There is no need to pass service IDs between layers via committed files or shared variables.

```terraform
# The environment layer looks up services by name at plan time
data "railway_service" "backend" {
  project_id = data.railway_project.main.id
  name       = "backend"
}

# Use the discovered ID directly
resource "railway_service_instance" "backend" {
  service_id     = data.railway_service.backend.id
  environment_id = railway_environment.this.id
  source_repo    = "myorg/my-app"
}
```

The two layers are fully decoupled. They share nothing except the project name.

## Using OpenTofu Workspaces for Environments

Use workspaces to manage multiple environments from a single configuration. The workspace name drives per-environment settings via a locals map:

```terraform
locals {
  env = terraform.workspace

  config = {
    dev = {
      branch       = "main"
      domain       = "dev.my-app.com"
      vcpus        = 0.5
      memory_gb    = 0.5
      source_image = "postgres:17.5-alpine"
    }
    prd = {
      branch       = "prd"
      domain       = "my-app.com"
      vcpus        = 1
      memory_gb    = 1
      source_image = "postgres:17.5-alpine"
    }
  }

  env_config = local.config[local.env]
}
```

Then select the workspace before planning:

```bash
tofu workspace select -or-create dev
tofu plan
tofu apply
```

## CI/CD Integration

Both layers are triggered manually — infrastructure changes are deliberate, not automated.

**Code deployments need no pipeline.** Once Railway is connected to a GitHub repo with a branch, it auto-deploys on push. The infrastructure pipeline only runs when you need to change the infrastructure itself (add a service, change resource limits, rotate a secret, add a domain).

See `examples/workflows/` in this repository for complete GitHub Actions workflow examples.

## Summary

| Concern | Services Layer | Environment Layer |
|---------|---------------|-------------------|
| **What it creates** | Project + empty services | Environments, service instances, volumes, variables, domains |
| **Source images/repos** | Never | Always (via `railway_service_instance`) |
| **How often it runs** | Once (or when adding a service) | Per environment, when config changes |
| **State isolation** | Single state | One workspace per environment |
| **Token required** | Account token | Account token |
| **Discovers services via** | — | Data sources (`data.railway_service`) |
