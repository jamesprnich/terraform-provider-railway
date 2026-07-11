## 0.11.1

### Fixes

Three categories of Railway transient errors now retry across a bounded window instead of failing immediately. Each fix has unit-test coverage of its classifier and retry mechanic; all were validated against a live Railway workspace before release.

* **Inline volume readback classification.** The post-create readback of an inline `volume` block on `railway_service` synthesised a `"not yet visible"` error whose string did not match `isNotFound`'s dictionary. `retryReadAfterCreateContext` misclassified it as terminal and bailed in a single poll interval. The sentinel is now wrapped in `NotFoundError` so `isNotFound` picks it up via `errors.As`, and the budget is bumped 30 s → 90 s to cover Railway's observed >28 s tail. Removes the v0.11.0 known limitation about inline volume rename.
* **Redeploy-in-flight conflict.** `variable_collection` Create/Update/Delete, `variable` Create/Update/Delete, and `service_instance` Update now retry when Railway returns `"Cannot redeploy yet, please wait for the original deployment to finish building"`. Delete paths downgrade to a Warning if the 3-minute budget expires so a still-building service cannot wedge `tofu destroy`; Create/Update paths hard-fail on timeout.
* **Volume creation throttle.** `railway_volume` Create and inline volume Create in `railway_service` both retry on Railway's per-mutation throttle (`"Whoa there pal! You are creating volumes too quickly. Try again in a sec"`), bounded to 60 s.

### Test hardening

* **Live lifecycle acceptance test now asserts `unmergedChangesCount == 0`** on every fork it creates. The C1 "deploys, not staged" property was previously defended only by the `StageInitialChanges: false` code setting; the assertion turns that from "we set the flag" into "we watched the flag's effect."
* **Manual comprehensive test regime** at `workshop/manual-test-regime/`. Tiered (least → most demanding), strictly sequential, workspace-hygiene checked. Not for CI — a human runs this against a real Railway workspace before shipping. Twelve self-contained test configs cover every non-skipped resource. Full run ~60–70 min, ~$0.10–$0.30 Railway compute.
* **Release workflow now runs the full CI pipeline** (`lint`, `unit`, `build`) before goreleaser publishes. Previously the release trusted that CI had already run on the merged commit; if a merge-squash introduced a regression, the release would still fire.

## 0.11.0

### BREAKING

* **`railway_service` is now a per-environment shell.** These fields moved off `railway_service` — they belong on `railway_service_instance`, which is the resource Railway's own API canonically models per environment: `source_image`, `source_image_registry_username`, `source_image_registry_password`, `source_repo`, `source_repo_branch`, `root_directory`, `config_path`, `cron_schedule`, `regions`. All were previously set via env-less GraphQL mutations (`serviceConnect`, `updateServiceInstance`) which create source connections across every non-fork environment in the project — a real bug when a project had multiple environments. Migration: move these attributes from any `railway_service` resource onto a matching `railway_service_instance` (create one per environment). No state migration is provided; delete affected resources from state before applying.
* **`railway_service.environment_id` added.** Under `strict_env_scoping = true` (provider default) it is required and RequiresReplace. Passing a fork environment scopes the service to that environment only. Omitting it under permissive mode (`strict_env_scoping = false`) restores the pre-v0.11.0 project-wide creation semantics.
* **`railway_environment.source_environment_id` added.** Under strict env-scoping it is required — every additional environment must be a fork of another. Non-fork environments break the safety property (see below); strict mode rejects them at plan time. Passing `false` on the provider block opts out.
* **`serviceDelete` mutation now accepts `environmentId`.** When `railway_service.environment_id` is set on the resource, `Delete()` passes it so the service is removed only from that fork. Legacy env-less deletes still work when the attribute is unset.

### Known Limitations

* Inline `volume` block on `railway_service` currently fails when Railway auto-assigns the same name as `volume.name` (e.g. `mount_path = "/var/lib/postgresql/data"` triggers `pgdata` auto-name that collides with the requested `name = "pgdata"`). Workaround: use the standalone `railway_volume` resource instead — that path is exhaustively tested and gives better lifecycle control. The schema description on `railway_service.volume` documents this. **Fixed in v0.11.1.**

### Security

* **Bump Go 1.25.11 → 1.25.12** to fix `crypto/tls` [GO-2026-5856](https://pkg.go.dev/vuln/GO-2026-5856) — "Invoking Encrypted Client Hello privacy leak." The provider's HTTP client used the affected paths (`providerserver.Serve`, `authedTransport.RoundTrip`). All CI workflows read the Go version from `go.mod`, so the bump cascades automatically.

### Enhancements

* **New `strict_env_scoping` provider attribute** (Bool, default `true`). Also settable via `RAILWAY_STRICT_ENV_SCOPING` env var. When enabled, forces `railway_service.environment_id` and `railway_environment.source_environment_id` to be set — the provider makes the class of bug that motivated this release structurally impossible to express in HCL. Set to `false` to opt out — you own the leak surface.
* **Plan-time diagnostics** for strict-mode violations via `ModifyPlan` — `tofu plan` fails with a clear error before any live mutation is attempted. Previously the same check was in Create and only fired at apply time.
* **Non-fork target rejection** — under strict mode, `railway_service.environment_id` pointing at a non-fork environment is rejected in Create with a specific diagnostic. Without this, Railway silently ignores the target id and creates the service across every non-fork environment in the project.
* **New `railway_service.icon` attribute** (String, Optional). Cosmetic icon displayed in the Railway dashboard. Applies project-wide (this is a genuinely service-level field on Railway's Service type, not per-environment).
* **`railway_volume` now retries the post-create read** with a 30s eventual-consistency budget. Fixes intermittent `"volume instance {id} not found"` failures that broke the first apply of every new environment when a volume was declared inline on `railway_service`.
* **Provider-side cooldown retry** on `projectCreate` and `environmentCreate`. Railway enforces "1 project per 30 seconds" and "one environment per user per 30 seconds" cooldowns; the provider now transparently waits them out with a 90s budget, so back-to-back applies no longer need external sleeps.
* **Inline volume post-create retry** — `getAndBuildVolumeInstance` in `railway_service.Create` is now wrapped in the same 30s eventual-consistency retry as `railway_volume.Create`. Prevents "inconsistent result after apply" when Railway's list endpoint hasn't caught up to the just-created volume.
* **Explicit `stageInitialChanges: false`** on `environmentCreate` — changes commit immediately rather than sitting as unmerged changes the user has to click "apply" on in the Railway dashboard.
* **`getAndBuildVolumeInstance` uses the service's own `environment_id`** rather than always resolving `defaultEnvironmentForProject`, so an inline volume on a fork-scoped service is read from its own environment.
* **Documented Railway API footguns** on the affected schema attributes:
  * `railway_service.name` — service names are unique per project, not per environment; use an environment prefix (e.g., `dev-backend`, `prd-backend`) when running the same role in multiple environments.
  * `railway_service.environment_id` — must be a fork; `depends_on = [railway_environment.<name>]` required because Terraform cannot infer the dependency from `project_id` alone.
  * `railway_environment.source_environment_id` — never fork a real environment; Railway's fork semantic copies every service, volume, variable, and configuration.

### Safety property

With this release, the "empty core" pattern is a first-class property enforced by the provider:

1. Project's default environment (`railway_project.default_environment.name = "core"`) stays empty forever. It is the project's only non-fork environment.
2. Every additional environment is a fork of `core` via `source_environment_id`.
3. Every service is scoped to a fork via `environment_id`.

Under this layout, any accidentally-unscoped `serviceCreate` lands inertly in `core` — it cannot contaminate a real environment. Strict env-scoping makes this the default; permissive mode restores the pre-v0.11.0 behaviour where an unscoped `serviceCreate` creates the service across every non-fork environment.

### Removed

* `serviceConnect` / `serviceDisconnect` / env-less `updateServiceInstance` mutations removed from the generated GraphQL client — they were project-wide and unused after the source-attachment path moved to env-scoped `serviceInstanceUpdate`.

## 0.10.0

### Enhancements
* Add `registry_credentials` block to `railway_service_instance` — enables deploying private Docker images (e.g. GHCR) by supplying `username` and `password` credentials. The `password` attribute is `Sensitive` and write-only (sent to Railway on create/update; never returned on read). Only available on Railway Pro plan.

## 0.9.0

### BREAKING
* **Removed `railway_webhook` resource.** Railway has removed the `webhookCreate`, `webhookUpdate`, and `webhookDelete` mutations from its public GraphQL API. Webhooks are now one channel type of the more general `notificationRule*` mutations. Use the new `railway_notification_rule` resource instead. Migration: delete any `railway_webhook.X` resources from state with `tofu state rm`, then create equivalent `railway_notification_rule.X` resources.

### Enhancements
* Bump Go from 1.25.0 → 1.25.8 (security patches for html/template, net/http, net/mail, syscall)
* Bump `terraform-plugin-testing` from v1.15.0 → v1.16.0
* Bump OpenTofu CI pin from 1.9.0 → 1.11.8 (HTTP/2 security fix)
* Refresh GraphQL schema from Railway API (2025-05-01 → 2026-05-15)
* Add `railway_notification_rule` resource — webhook, Slack, email and other notification channels (replaces `railway_webhook`)
* Add `railway_project_token` resource — project-scoped deploy tokens for CI/CD pipelines
* Add `railway_trusted_domain` resource — workspace-level trusted domain for SSO
* Add `railway_bucket` resource — S3-compatible object storage bucket
* Add `railway_ssh_public_key` resource — SSH public key for workspace
* Add `railway_project_member` resource — full membership CRUD (Add mutation added by Railway)

### Known Limitations
* `railway_bucket` Delete is a no-op — Railway has not exposed a `bucketDelete` mutation. `tofu destroy` removes the bucket from state only; the bucket persists in Railway until project deletion or manual cleanup via the dashboard.

## 0.8.0

### BREAKING
* Volume import format changed from `project_id:volume_id` to `project_id:volume_id:service_id:environment_id`
* Webhook import format changed from `webhook_id` to `project_id:webhook_id`

### Enhancements
* Add `railway_webhook` resource — HTTP webhook notifications for project events
* Add `railway_egress_gateway` resource — static egress IP for external service allowlisting
* Add `railway_private_network` resource — private network for internal service-to-service communication
* Add `railway_private_network_endpoint` resource — connects a service to a private network with DNS name
* Add `railway_deployment_trigger` resource — auto-deploy from GitHub/GitLab on push (re-added after v0.5.0 removal)
* Add `railway_volume_backup_schedule` resource — automatic backup schedules for volume instances
* Add `data.railway_project` data source — look up project by ID or name
* Add `data.railway_environment` data source — look up environment by ID or name
* Add `data.railway_service` data source — look up service by ID or name
* Add environment rename support — `railway_environment` name changes no longer force destroy/recreate
* Add custom domain target port update — `railway_custom_domain` target_port changes no longer force destroy/recreate
* Upgrade `terraform-plugin-framework` from v1.2.0 to v1.19.0
* Add `UseStateForUnknown` plan modifier to all stable Computed attributes across all resources
* Add schema version tracking (`schema_version.go` + `scripts/check-schema.sh`)
* Add documentation for all 17 resources and 3 data sources

### Bug fixes
* Fix `railway_custom_domain` panic on empty DNS records from API
* Fix `railway_project` Create orphaning resources when environment count is unexpected
* Fix `railway_webhook` ImportState not setting `project_id` (Read would fail after import)
* Fix `railway_egress_gateway` Delete failing when resource already deleted externally
* Fix `railway_private_network_endpoint` Delete failing when resource already deleted externally
* Fix `isNotFound` matching — add `"not found"` pattern for Railway API error messages like `"Project not found"`
* Fix `railway_environment` Go struct field typo (`ProjecId` → `ProjectId`)
* Fix `railway_environment` Read not detecting deleted environments (Railway returns null, not an error)
* Fix `railway_volume` import fragility — null environment/service matching accepted any volume instance
* Fix `railway_service` inline volume creation — pass explicit `environmentId` to avoid Railway "deploy to all environments" failure on new services
* Fix `railway_service` inline volume creation — use local `&serviceId` variable instead of `ValueStringPointer()` for reliable pointer semantics
* Fix `railway_service` inline volume plan modifiers — replace `UseStateForUnknown()` with custom `useStringStateForUnknownIfNonNull()` / `useFloat64StateForUnknownIfNonNull()` to prevent "inconsistent result after apply" when adding volume to existing service
* Fix `railway_service` Create — reorder source connection (image/repo) before volume creation for API stability
* Fix `railway_service` Create — set computed fields (regions, volume) to null instead of unknown before early state save
* Fix `railway_variable_collection` ID instability — changed ID format from `serviceId:envId:NAME1:NAME2:...` to `serviceId:envId` so variable name changes don't break state
* Fix `railway_environment` Read using stale `getEnvironment(id)` query — switched to authoritative `getEnvironments(projectId)` list which correctly reflects deletions
* Fix `railway_environment` ImportState not setting `project_id` (Read would fail after import)
* Fix `railway_environment` Delete failing when environment already deleted externally — added pre-delete existence check via project environment list
* Fix `railway_service_domain` Delete failing with "operation already in progress" when concurrent deletes occur
* Fix `railway_custom_domain` Delete failing with "operation already in progress" when concurrent deletes occur
* Fix `railway_tcp_proxy` Delete failing with "operation already in progress" when concurrent deletes occur
* Fix `railway_tcp_proxy` domain field inconsistency — normalize trailing dot between Create and Read API responses
* Fix `railway_service` inline volume orphan leak — when volume rename fails after creation, the orphaned volume is now cleaned up automatically (both Create and Update paths)
* Fix all Delete methods — introduced `isNotFoundOrGone()` for Delete-only use, matching Railway's non-standard "Not Authorized" and "Problem processing request" responses for already-deleted resources. `isNotFound()` remains narrow (safe for Read methods where false positives would silently remove live resources from state)
* Fix `railway_deployment_trigger` acceptance tests — corrected GitHub repo name from `railway-terraform-provider` to `terraform-provider-railway`
* Add `volume_instance_id` computed attribute to `railway_volume` — enables chaining to `railway_volume_backup_schedule` (previously the volume ID was used where the volume instance ID was required)
* Fix `railway_service` root_directory description typo — "Directory to user" → "Directory to use"
* Fix `docs/resources/webhook.md` example filter format — changed `["DEPLOY"]` to `["deploy.completed", "deploy.started"]`
* Fix `docs/resources/custom_domain.md` — add missing `target_port` Optional field documentation
* Fix `railway_service` inline volume creation — unknown computed sub-fields (`id`, `size`) in early state save caused "Provider returned invalid result object after apply" when volume creation failed
* Fix `railway_service` inline volume creation — add retry with backoff for Railway API "Problem processing request" errors due to eventual consistency on newly created services

## 0.7.0

### Enhancements
* Add `railway_volume` resource — standalone volume with environment-specific targeting, replacing the default-environment-only `volume` block on `railway_service`
* Add `railway_service_instance` resource — per-environment service configuration including source, build, deploy settings, and resource limits (vCPUs, memory)
* Add `serviceInstanceLimitsUpdate` GraphQL mutation support for setting CPU and memory limits
* Add `serviceInstanceUpdate` with environment targeting (previously hardcoded to null/all environments)

## 0.6.1

### Bug fixes
* Fixes issue with volume creation

## 0.6.0

### BREAKING
* Rename `team_id` to `workspace_id` in `railway_project`

## 0.5.2

### Enhancements
* Added validation for checking `cron_schedule` and replica count

## 0.5.1

### Bug fixes
* Fixes issue with updating service regions defaulting to `us-west1`

## 0.5.0

### BREAKING
* Remove `railway_deployment_trigger` resource
* Remove `region` from `railway_service` in favor of `regions` multi region support

### Bug fixes
* Fixes issue with tcp proxy not being used in service after creating
* Fixes issue with updating project

## 0.4.6

### Bug fixes
* Fix issue with `targetPort` for custom domains being set to `0` if not provided
* Changed reading custom domain from service instance instead of using id

## 0.4.5

### Bug fixes
* Fix issue with optional `team_id` in `resource_project`
* Fix issue with `region` and `volume` in `resource_service`

## 0.4.4

### Bug fixes
* Fix issue with setting `source_image_registry_*` in `resource_service`

## 0.4.3

### Enhancements
* Added `railway_variable_collection` resource

## 0.4.2

### Bug fixes
* Fix issue with root directory and config path not being read correctly

## 0.4.1

### Enhancements
* Added `region` to `railway_service`
* Added `num_replicas` to `railway_service`
* Added registry credentials to `railway_service`

## 0.4.0

### BREAKING
* Add required `source_repo_branch` to `railway_service` when `source_repo` is present

## 0.3.1

### Bug fixes
* Fix issue with replicas of a service being set to `0`

## 0.3.0

### BREAKING
* Remove `railway_plugin` resource
* Remove `railway_plugin_variable` data source

### Enhancements
* Add `railway_tcp_proxy` resource
* Support `volume` in `railway_service` resource

## 0.2.0

### Enhancements
* Add `railway_service_domain` resource
* Add `railway_custom_domain` resource

### BREAKING
* Remove `project_id` input from `railway_deployment_trigger`
* Remove `project_id` input from `railway_variable`

## 0.1.2

### Enhancements
* Add `railway_deployment_trigger` resource

## 0.1.1

### Enhancements
* Add support for more service settings

## 0.1.0 (First release)
