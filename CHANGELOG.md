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
