## 0.11.4

### Fixes

* **`railway_custom_domain` silently discarded the TXT verification record and could substitute it for the CNAME target on any Railway-side reorder.** Railway returns `status.dnsRecords` as an unordered array containing (at least) the CNAME that routes traffic and the TXT that proves domain ownership. The provider selected the traffic-routing CNAME by taking `dnsRecords[0]` — correct only by luck of ordering, and the TXT verification record was unreachable through the resource at all. Consumers had to shell out to the Railway GraphQL API (or run an external data source through the provider token) to fetch the TXT value themselves. Two changes:
  1. The GraphQL fragment now selects `recordType` and `purpose`; the CNAME is picked by `purpose == TRAFFIC_ROUTE` with a fallback to `recordType == CNAME`. Order-independent by construction — `dns_record_value` is the CNAME target regardless of how Railway serialises the record list. See `selectTrafficRouteCNAME` in `internal/provider/resource_custom_domain.go` and the 6-case `TestSelectTrafficRouteCNAME_pure` unit test.
  2. Three new computed attributes expose Railway's dedicated verification fields directly (rather than reconstructing them from the record list):
     * `verified` (Bool) — `status.verified`. `true` once Railway's verifier confirms the TXT is in place.
     * `verification_dns_host` (String) — `status.verificationDnsHost`. The hostname the user places the TXT record at (e.g. `_railway-verify.dev.example.com`).
     * `verification_token` (String) — `status.verificationToken`. The value that goes into the TXT record.

  Consumers can now read the verification pair straight off the resource — the whole `data "external"` + shell-script + secret-piping dance is unnecessary.

* **`railway_egress_gateway` picked the wrong region under multi-region deploys.** `Read` took `EgressGateways[0].Region` unconditionally. For a service with egress gateways in more than one region, the provider would silently overwrite `region` state to whichever gateway happened to be first in Railway's response, causing spurious `RequiresReplace` on the next plan. Fixed at `resource_egress_gateway.go` — selection now matches by `gw.Region == state.Region.ValueString()`, never by array position. Same class of bug as the custom-domain issue above; caught via a scope-check grep for `\.[A-Z][a-zA-Z]+\[0\]` across `internal/provider/`.

### Tests

* **`TestSelectTrafficRouteCNAME_pure`** — pure-Go table test hitting the selector across six cases: empty array, CNAME with TRAFFIC_ROUTE purpose, reversed order, purpose fallback to recordType, only-TXT array (returns nil), and a defensive mislabeled-record case (asserts purpose takes precedence over recordType).
* **`TestCustomDomainResource_cnameSelectionIsOrderIndependent`** — resource-level mock test that drives Read against both `[CNAME, TXT]` and `[TXT, CNAME]` orderings and asserts `dns_record_value` picks the CNAME in both. Reporter's specific concern.
* **`TestCustomDomainResource_cnameFallsBackToRecordType`** — asserts the `purpose == UNSPECIFIED` fallback correctly picks the CNAME by `recordType`.
* **`TestAccCustomDomainResourceDefault` extended** to assert the three new verification attributes are populated by a real Railway workspace (`verified == false` for a fresh unverified domain, `verification_dns_host` and `verification_token` both set). Live-verified: Railway does populate these fields on the customDomain response — the design choice (use dedicated status fields over parsing dnsRecords) is validated end-to-end.

## 0.11.3

### Fixes

* **`railway_service_instance` was unloadable against any v0.11.1-or-earlier state.** v0.11.2 changed `pre_deploy_command` from `list(string)` to `string` and correctly bumped the resource schema Version 1 → 2, but shipped without an `UpgradeState` implementation. Every user with existing state hit the framework's `Unable to Upgrade Resource State — was expecting an implementation for version 1 upgrade` error on the first refresh, with no recovery short of downgrading or hand-editing state. Fixed by implementing `UpgradeState` on `ServiceInstanceResource` — see `internal/provider/resource_service_instance_upgrade.go`. The v1→v2 upgrader converts `pre_deploy_command` list values: `null → null`, single element → string, multi-element → `strings.Join(..., " && ")` (lossless per the reporter's recommendation, though the provider never produced multi-element state itself). Anyone on v0.11.1 can now upgrade directly to v0.11.3 with no manual state surgery.

* **`railway_service` and `railway_environment` had the same latent bug from v0.11.0.** Both bumped their schema Version 1 → 2 in v0.11.0 as part of the strict env-scoping redesign, and both shipped without `UpgradeState`. The v0.11.0 CHANGELOG documented a manual `terraform state rm` migration path, but users who missed that guidance hit the framework's cryptic "was expecting an implementation for version 1 upgrade" error with no explanation of what to do. v0.11.3 registers rejecting upgraders on both resources that surface the actual migration instructions in a clear diagnostic. Users who followed the v0.11.0 CHANGELOG migration are unaffected (their state is already at Version 2).

### Prevention

The class of bug above is systemic — the terraform-plugin-framework does not validate `Version` vs `UpgradeState` at registration time; the mismatch only surfaces at the first refresh RPC after a real user's state hits the mismatched provider. `tfproviderlint` has no rule for it either. Mock unit tests, live acceptance tests, and code review all missed v0.11.2 because none of them exercise state carrying a prior-version stamp. v0.11.3 adds three guardrails that would have blocked v0.11.2 in CI:

* **`TestSchemaVersionsHaveUpgraders`** — a reflection-based invariant test that walks every registered resource and asserts that if current `Version ≥ 2`, the resource implements `ResourceWithUpgradeState` and its `UpgradeState` returns entries for every prior version in `[1..Version-1]`. Runs as part of `go test ./internal/provider/` — no new CI job. This is a novel test — no other Terraform provider ships it because everyone else has the same latent gap; consider surfacing upstream. See `internal/provider/schema_upgrade_invariant_test.go`.
* **HashiCorp-pattern per-upgrader unit test** (`TestUpgradePreDeployCommandV1ToV2`) — hand-builds prior-shape values, calls the upgrader directly, `Equal`-compares against expected new-shape values. Seven cases: null, unknown, empty, single, two-element, three-element, shell-metachar-containing. Same shape as `internal/service/kinesis/migrate_test.go` in `hashicorp/terraform-provider-aws`.
* **Generated-code drift check in CI** — `test.yml` and `release.yml` now run `go run github.com/Khan/genqlient` and fail if `internal/provider/generated.go` diverges from the schema. Canonical HashiCorp idiom.
* **CHANGELOG entry required for tagged version** — `release.yml` now greps `CHANGELOG.md` for the tag version before goreleaser publishes. Fails with a clear message if the entry is missing.

## 0.11.2

### BREAKING

* **`railway_service_instance.pre_deploy_command` is now `string`, not `list(string)`.** Railway's underlying API models the field as `[String!]` but its dashboard exposes a single command input and its server-side validation rejects lists with more than one element (`Error in preDeployCommand - Invalid input`). The provider now reflects the dashboard shape so invalid configurations fail at plan time rather than apply time. Migration: change `pre_deploy_command = ["python manage.py migrate"]` to `pre_deploy_command = "python manage.py migrate"`. The provider's schema Version bumps 1 → 2; no state upgrader is provided because the previous shape was unusable at runtime (see fix below) — no state file exists with a real value.
* **`railway_service` import id now accepts `<service_id>:<environment_id>`** for fork-scoped services, and rejects the bare `<service_id>` form under strict env-scoping (provider default). A bare id would leave `environment_id` null in state; the subsequent plan would see the fork env_id in HCL as a change requiring replace and silently destroy the just-imported service. Under permissive env-scoping (`strict_env_scoping = false`), both forms remain valid — the bare form imports a project-wide service. Migration: any prior `tofu import railway_service.foo <svc>` invocation was broken under v0.11.0/v0.11.1 (import failed ImportStateVerify because `environment_id` was never populated); use the compound form going forward.

### Fixes

* **`railway_service_instance.pre_deploy_command` was unreadable and poisoned every service instance that set it.** Railway's GraphQL schema declares the field as `JSON` on the read type (loose scalar) but `[String!]` on the write input. genqlient's global JSON→`map[string]interface{}` binding was applied to the read side, so `Read` panicked with `json: cannot unmarshal array into Go struct field ... of type map[string]interface {}` the moment Railway returned a non-null command list. Because the WRITE succeeded before Read fired, Railway retained the value and every subsequent `refresh`/`plan` hit the same panic — the resource became permanently unplannable, recoverable only by clearing the field manually in the dashboard or downgrading. Fixed by pinning a per-field bind of `*[]string` on the read type (`internal/provider/resource_service_instance.graphql:25`); the global JSON binding is untouched, so `DeploymentMeta`, `EnvironmentConfig` and other genuinely object-shaped fields are unaffected. Scope check: `preDeployCommand` is the only same-class bug across the schema — every other read-side JSON field is either genuinely object-shaped, on an input type, or not queried by the provider.
* **`railway_service` import lost `environment_id` on v0.11.0/v0.11.1.** `ServiceResource.ImportState` used `ImportStatePassthroughID` and never set the env_id first-class attribute added by v0.11.0. Any import round-trip on a fork-scoped service failed `ImportStateVerify` because the imported state didn't match the HCL config. Fixed with a compound-id parser (see BREAKING) — see `parseServiceImportId` in `internal/provider/resource_service.go` and the nine-case `TestParseServiceImportId` unit test.
* **Standalone `railway_volume` creation was flaky under workspace load.** Post-create readback used a 30 s retry budget, but Railway's list endpoint has been observed exceeding a 28 s eventual-consistency tail — the inline-volume path bumped its budget 30 s → 90 s in v0.11.1 for exactly this reason. Applied the same bump to `railway_volume` (`internal/provider/resource_volume.go:217`). Standalone volumes and inline volumes share Railway's upstream indexing path, so they share the same tolerance.

### Tests

* **Three mock unit tests** for `railway_service_instance.pre_deploy_command`: `TestServiceInstanceResource_preDeployCommand_readSucceeds` (proves the JSON unmarshal fix by returning `preDeployCommand: ["migrate"]` in the read response and asserting the update input serialises as a one-element list on the wire); `TestServiceInstanceResource_preDeployCommand_lifecycle` (Create → Update → Read-after-Update round-trip that would panic pre-fix); `TestServiceInstanceResource_imageSource_lifecycle` (comprehensive image-sourced coverage with `registry_credentials` + `restart_policy_*` + `healthcheck_*` + `pre_deploy_command` all set together, drives Create → Update → PlanOnly, proves the write-only password is preserved from state on Read so there is no perpetual diff).
* **Pure-Go unit test** `TestParseServiceImportId` covers the nine edge cases of the new compound-id parser: compound form under both strict and permissive modes, bare id under permissive, bare id rejected under strict, empty string, colon prefix with empty service_id, trailing colon with empty environment_id (rejected as malformed under both modes rather than misclassified as a strict-mode omission), and an environment_id half that contains further colons (preserved verbatim).
* **Lifecycle acceptance test extended.** `TestAccLifecycle_forkTopology` now sets `pre_deploy_command` on the fork-scoped service in step 1 (exercises Read-after-Create — the direct bug trigger) and updates it in step 2 (exercises Read-after-Update — the reporter's specific concern). `TestAccServiceResource_basic` now imports with the compound `<service_id>:<environment_id>` form via a new `testAccServiceImportStateIdFunc` helper.
* All three fixes plus the shape change validated against a live Railway workspace before release: 43 acceptance tests pass, 7 skip (all documented workspace-scoped skips), 0 failures.

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
