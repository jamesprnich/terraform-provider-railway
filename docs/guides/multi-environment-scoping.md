# Multi-Environment Scoping

Railway's default behaviour is that a service created without an environment id
lands in **every environment that isn't a fork** — an intentional API property
so that the dashboard "add service to project" gesture propagates to every
environment. In Terraform, that default silently multiplies your intent: a
single `railway_service` resource can materialise as one service in `dev`, one
in `tst`, one in `qa`, one in `prd`, all sharing state that nobody asked to
share. When two teams then deploy to different envs the same day, the effects
race — and the losing team spends an afternoon trying to work out why their
production Postgres crashed.

The v0.11.0 provider introduces **strict environment scoping** as the default,
along with a two-attribute mechanism that makes the correct pattern the
easiest one to express. This guide explains what changed, why the design has
the shape it has, and how to consume it — including the few footguns Railway
retains at the API level that the provider surfaces in schema descriptions.

## The problem in one sentence

Prior to v0.11.0, an unscoped `railway_service` created a `Service` record
that Railway propagated to every non-fork environment in the project, and
subsequent `serviceConnect` / `updateServiceInstance` mutations happened
without an environment context, so their effects propagated the same way.

## The design

The mechanism relies on a distinction Railway itself makes: an environment is
either a **fork** of another environment (its `sourceEnvironment` is non-null)
or a **non-fork** (its `sourceEnvironment` is null). Railway's own field
description for `ServiceCreateInput.environmentId` says the quiet part out
loud:

> Environment ID. If the specified environment is a fork, the service will
> only be created in it. Otherwise it will [be] created in all environments
> that are not forks of other environments.

The provider organises resources so that the "otherwise" branch is never
what the user wanted:

1. The project's default environment (its `defaultEnvironmentName`, honoured
   only at create time — set it to `core`) is the project's only non-fork
   environment. It is **kept empty forever**. No services, no volumes, no
   variables. It exists purely so the fork mechanism has an anchor.
2. Every other environment (`dev`, `tst`, `qa`, `prd` — whatever your
   deployment topology names them) is created as a **fork** of `core` via
   `source_environment_id`.
3. Every service, volume, and variable specifies the fork it belongs to via
   `environment_id` (or `environmentId` on the child resource — service
   instances, volumes, variables all already do this).

Under this arrangement, `core` is the project's only non-fork environment.
Any `serviceCreate` that forgets to scope itself lands inertly in `core`,
which has no configuration to inherit and no traffic to disrupt. Every other
mutation is fork-scoped by construction. **The safe outcome becomes the
default outcome, rather than a convention someone must remember.**

The two attributes that carry the design:

- `railway_environment.source_environment_id` — required under strict
  env-scoping. Set it to `railway_project.<name>.default_environment.id` and
  the resource becomes a fork of your empty `core`.
- `railway_service.environment_id` — required under strict env-scoping. Set
  it to a fork environment's id.

Under `strict_env_scoping = true` (the provider default), omitting either
attribute is a plan-time error. The rejection happens before any live
mutation — you see the diagnostic on `tofu plan`, not on apply.

## A worked example

```hcl
provider "railway" {
  # strict_env_scoping defaults to true; leave it that way.
}

# Empty non-fork default environment named `core`.
resource "railway_project" "app" {
  name = "my-app"

  default_environment = {
    name = "core"
  }
}

# dev — a fork of the empty `core`. Everything created here stays here.
resource "railway_environment" "dev" {
  name                  = "dev"
  project_id            = railway_project.app.id
  source_environment_id = railway_project.app.default_environment.id
}

# The service — an empty shell scoped to dev. Note the depends_on: railway_service
# references only project_id, so Terraform cannot infer the dependency on
# railway_environment.dev, and without depends_on the environment may not
# exist when serviceCreate runs.
resource "railway_service" "backend" {
  name           = "dev-backend"
  project_id     = railway_project.app.id
  environment_id = railway_environment.dev.id
  depends_on     = [railway_environment.dev]
}

# Source, build, deploy configuration — env-scoped, on the instance.
resource "railway_service_instance" "backend" {
  service_id     = railway_service.backend.id
  environment_id = railway_environment.dev.id
  source_image   = "myorg/backend:1.2.3"
  vcpus          = 0.5
  memory_gb      = 0.5
}
```

To grow this to `dev + tst + qa + prd`, duplicate the `railway_environment`
+ `railway_service` + `railway_service_instance` block per environment (or
use a `for_each`). Every service name gets prefixed with its environment
(`dev-backend`, `tst-backend`, …) because service names are unique per
project, not per environment.

## The four Railway-side footguns v0.11.0 documents

None of these are provider bugs. All of them are properties of the Railway
API that catch people out; the provider documents them on the affected
schema attribute and, where useful, enforces them at plan time.

### 1. `sourceEnvironmentId` copies everything from the source

Railway's fork mutation copies every service, volume, variable, and
configuration from the source environment. This is the intended semantic
when you want to duplicate a working environment into a new one, but it
turns `railway_environment.prd { source_environment_id = railway_environment.dev.id }`
into a landmine: you've just duplicated dev into prd, with all its
credentials, without the config asking to.

**Always fork the empty `core` environment.** The provider warns about this
in the schema description on `source_environment_id`.

### 2. `environmentId` is silently ignored when the target is a non-fork

Setting `railway_service.environment_id` to a non-fork environment does not
scope the service to that environment. Railway falls back to its default
"create across every non-fork environment" behaviour, ignoring the id
entirely.

The provider catches this under strict mode: `Create` looks up the target
environment and rejects the service if the target has no
`source_environment_id`. Under permissive mode, no check is performed — you
opted in to Railway's raw semantics.

### 3. Service names are unique per project, not per environment

`serviceCreate` rejects a name that already exists in the project regardless
of which environment the new service is scoped to. Two services named
`backend`, one in `dev` and one in `prd`, cannot coexist. The convention
(also the provider's recommendation) is to prefix every service name with
its environment: `dev-backend`, `prd-backend`. This also matches how
Railway's private DNS works — the DNS name is just the service name.

**The same uniqueness rule applies to `railway_volume.name`** — volume names
are also unique per project. Use the same env-prefix convention:
`dev-postgres-data`, `prd-postgres-data`. The `examples/workflows/main.tf`
reference example demonstrates this end-to-end.

**Environments themselves do not need a prefix** — the environment name IS
the prefix source. Use short, canonical names: `dev`, `tst`, `qa`, `prd`.

### 4. `railway_service` cannot infer its environment dependency

The `railway_service` resource references only `project_id`. Terraform's
implicit dependency tracking cannot see the reference to `environment_id`
as a dependency on the corresponding `railway_environment` resource — the
service's provider schema doesn't declare `environment_id` as pointing at
`railway_environment`, it's just a string. Without an explicit
`depends_on = [railway_environment.<name>]`, the service can be scheduled
before the environment exists, and the create fails.

Every service should carry the `depends_on` — the schema description on
`environment_id` says so.

## The escape hatch

If you have a use case that genuinely wants Railway's default project-wide
behaviour, or you're intentionally testing what happens without scoping,
set the flag on the provider block:

```hcl
provider "railway" {
  strict_env_scoping = false
}
```

Under permissive mode:

- `environment_id` and `source_environment_id` behave as plain Optional
  attributes; omitting them is legal.
- The non-fork target check is skipped.
- Services created without `environment_id` land project-wide (Railway's
  "all non-fork environments" semantic).
- Environments created without `source_environment_id` are non-fork.

The flag can also be set via the `RAILWAY_STRICT_ENV_SCOPING` environment
variable — useful in CI where the same provider config is reused across
runs.

## Migrating from v0.10.0

v0.11.0 is a breaking release. The `railway_service` resource was reduced
to a shell — every configuration field was moved to `railway_service_instance`,
which is the resource Railway's API canonically models per environment. If
you're upgrading:

1. **Add `environment_id`** to every `railway_service` resource. Reference
   the environment the service was previously implicitly bound to.
2. **Add `depends_on`** on the same block, pointing at the environment
   resource.
3. **Move `source_image`, `source_repo`, `source_repo_branch`, `root_directory`,
   `config_path`, `cron_schedule`, and any registry credentials off**
   `railway_service` and onto the corresponding `railway_service_instance`.
   If a `railway_service_instance` doesn't already exist for that
   `(service, environment)` pair, add one.
4. **Add `source_environment_id`** to every additional `railway_environment`.
   For an existing project, point at the current default environment.
5. **Rename the default environment to `core`** — but note that this is
   honoured **only at project creation time**. On an existing project the
   name is fixed; you can adopt the pattern on new projects and use whatever
   the existing name is on the current one.

There is no state upgrader. If you were on v0.10.0, you'll need to
`terraform state rm` the affected resources and re-import them (or, if
you're starting from a fresh apply, just apply the new config).

## Reference — quick lookup

| Concern | Attribute | Enforcement |
|---|---|---|
| Force fork on additional envs | `railway_environment.source_environment_id` | Plan-time under strict; permitted null under permissive |
| Force scope on services | `railway_service.environment_id` | Plan-time under strict; permitted null under permissive |
| Reject non-fork target | (checked in `railway_service.Create`) | API preflight under strict; skipped under permissive |
| Opt out of all four checks | `strict_env_scoping = false` on the provider block, or `RAILWAY_STRICT_ENV_SCOPING=false` in the env | Applies to every resource in the config |
| Volume post-create retry | `railway_volume` (always on) | Unconditional — this fixes a bug and has no opt-out |
