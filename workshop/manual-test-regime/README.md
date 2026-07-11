# Comprehensive manual test regime

Human-triggered end-to-end validation of the Railway Terraform provider against a real Railway workspace. Not for CI/CD — run this before cutting a release, after material CRUD changes, or when Railway API behaviour is suspected to have shifted.

## What it does

Applies + destroys a tiered matrix of throwaway Railway projects, all prefixed `AAA-provctest-`, against the workspace named by the API token in `RAILWAY_TOKEN`. Every project is destroyed at the end of its own test; a post-flight step lists any project matching the prefix and fails the run if any survived.

## What it does NOT do

- Touch any resource not named `AAA-provctest-*`.
- Mutate workspace-level state (`railway_project_member`, `railway_ssh_public_key`, `railway_trusted_domain`).
- Attach a real DNS name (`railway_custom_domain` is skipped).
- Create anything without a delete path (`railway_bucket` is skipped — its Delete is a documented no-op).
- Run anything in parallel — every apply, every destroy, every tier is strictly serial.

## Prerequisites

1. **OpenTofu** ≥ 1.11 or **Terraform** ≥ 1.0 on PATH.
2. **Go** on PATH (used for Tier 0 build + unit tests, and for locating the provider binary).
3. **jq** on PATH (used by workspace hygiene checks).
4. **Compiled provider binary** at `$(go env GOPATH)/bin/terraform-provider-railway`. From the repo root:
   ```
   go build -o "$(go env GOPATH)/bin/terraform-provider-railway" .
   ```
5. **Railway API token** with permission to create projects on the target workspace.
6. **Workspace ID** for the target workspace (looked up once, needed by `railway_notification_rule`).

## Environment variables

| Var | Required | Purpose |
|---|---|---|
| `RAILWAY_TOKEN` | ✅ | Railway API token. Used by the provider and by workspace-hygiene checks. |
| `RAILWAY_TEST_WORKSPACE_ID` | ✅ | Workspace ID for `railway_notification_rule` in Tier 2. |
| `PROVIDER_BINARY_DIR` | ⬜ | Directory containing the compiled provider binary. Default: `$(go env GOPATH)/bin`. |
| `TEST_PREFIX` | ⬜ | Project-name prefix `check-workspace.sh` scans for. Default: `AAA-provctest-`. Only change if you also edit the config files' project names. |
| `LOG_DIR` | ⬜ | Where per-test logs land. Default: `./logs-<timestamp>`. |
| `TIERS` | ⬜ | Space-separated tier numbers to run. Default: `0 1 2 3 4`. Skip cheap tiers with e.g. `TIERS="3 4"`. |
| `COOLDOWN_SECONDS` | ⬜ | Sleep between tests / tiers. Default: `30`. Railway enforces a 30 s project-create cooldown per user, so shorter values will slow the run down via provider-side retries rather than speed it up. |

**Nothing workspace-specific is hard-coded in the configs.** The provider source is `jamesprnich/railway` (this repo's registry namespace); if running from a fork with a different namespace, edit the `required_providers` blocks in each `main.tf` and update `tofurc.template`.

## Running

```
export RAILWAY_TOKEN=<your-token>
export RAILWAY_TEST_WORKSPACE_ID=<your-workspace-id>
./run.sh
```

Full run wall clock: **~60–70 minutes**. Sequential, dominated by Railway's 30 s project-create cooldown.

To run a single tier:
```
TIERS="3" ./run.sh
```

To run without waiting for cooldowns (e.g. when you're on a fresh workspace and know the cooldowns won't fire):
```
COOLDOWN_SECONDS=0 ./run.sh
```

## What's in each tier (least → most demanding)

### Tier 0 — offline (~10 s)
- `go build ./...`
- `go vet ./...`
- Helper unit tests (`TestIsNotFound_*`, `TestRetryReadAfterCreateContext_*`, `TestIsRedeployNotReady*`, `TestRetryRedeployContext_*`)
- Pre-flight workspace check — refuses to continue if any `AAA-provctest-*` project already exists on the workspace.

### Tier 1 — cheap live (~8 min, ~30 GraphQL calls, $0 compute)
| Test | Exercises |
|---|---|
| `t1_1_project_crud` | `railway_project` — create, in-place rename, destroy |
| `t1_2_env_fork_nonfork` | `railway_environment` — fork under strict mode + non-fork under permissive mode in the same run, using provider aliases |
| `t1_3_data_sources` | `data.railway_project`, `data.railway_environment`, `data.railway_service` — round-trip by-id AND by-name |
| `t1_4_strict_plan_reject` | `ModifyPlan` — asserts strict mode rejects service without `environment_id` and env without `source_environment_id` at plan time |

### Tier 2 — non-compute resources (~15 min, ~120 GraphQL calls, $0 compute)
One project, one env, exercises every non-workspace resource that has a working Delete:
`railway_project`, `railway_environment`, `railway_service`, `railway_shared_variable`, `railway_variable`, `railway_variable_collection`, `railway_volume` (standalone), `railway_volume_backup_schedule`, `railway_service_domain`, `railway_tcp_proxy`, `railway_private_network`, `railway_private_network_endpoint`, `railway_egress_gateway`, `railway_project_token`, `railway_notification_rule`.

**No `railway_service_instance` = no billable compute.**

### Tier 3 — compute deploys (~15 min, ~40 GraphQL calls, ~$0.20 compute)
| Test | Exercises |
|---|---|
| `t3_1_fork_topology` | Empty `core` + two forks (`dev`/`prd`) + services scoped to each + standalone volume. Includes a live GraphQL assertion that services are scoped only to their fork and `core` is empty — the strict env-scoping safety property. |
| `t3_2_e2e_both_volume_paths` | Full workflow: 10 resources including `railway_service.postgres` with an **inline volume** + `railway_volume.app_cache` **standalone** in the same project. Applies + destroys immediately (no sleep) to stress the redeploy retry helper. |

### Tier 4 — stress + edge (~20 min, ~80 GraphQL calls, ~$0.10 compute)
| Test | Exercises |
|---|---|
| `t4_1_flake` × 5 | Baseline inline-volume config × 5 back-to-back iterations. Confirms the 90 s post-create readback retry budget still holds across Railway's tail. |
| `t4_2_rename_lifecycle` | `railway_environment.rename`, `serviceUpdate`, inline volume rename — three distinct mutations across one two-apply-then-destroy sequence. |
| `t4_3_collision` | Two services in one project both requesting the same volume name. Apply is EXPECTED to fail with Railway's real `A volume named "X" already exists in this project` error; destroy MUST still succeed. |
| `t4_4_rapid_cycle` × 2 | Full `service` + `service_instance` + `variable_collection` create + immediate destroy. Stresses the redeploy retry: the initial build is still in flight when destroy runs. |
| `t4_5_deployment_trigger` | `railway_deployment_trigger` against a public GitHub repo. Requires the workspace to have the Railway GitHub app connected. |

## Post-flight

Every run ends with a read-only GraphQL query listing every project whose name starts with `$TEST_PREFIX` (default `AAA-provctest-`). Any survivor fails the run.

`check-workspace.sh` can be run standalone at any time:
```
RAILWAY_TOKEN=... ./check-workspace.sh
```

## Skip list, with reasons

| Resource | Why skipped |
|---|---|
| `railway_project_member` | Workspace-level mutation — cannot be isolated |
| `railway_ssh_public_key` | User-global — cannot be isolated |
| `railway_trusted_domain` | Workspace-level mutation |
| `railway_custom_domain` | Needs real DNS |
| `railway_bucket` | Delete is a documented no-op (would leave orphans) |

## Adding a new test

1. Create `tests/<name>/main.tf`. Prefix any Railway resource name with `AAA-provctest-` so the workspace-hygiene check catches it if left behind.
2. If the test needs an in-place update or rename, express it with a `-var` that changes value between two applies (see `t1_1_project_crud` and `t4_2_rename` for the pattern).
3. Wire it into the relevant `TIER <n>` block in `run.sh` via `run_test` (creates + destroys), `run_plan_only_expect_fail` (plan-only + regex), or an inline block for multi-step scenarios.
4. Update the tier table above.
5. Dry-run: `TIERS="<n>" ./run.sh` and inspect logs.

## Known limitations

- Depends on Railway's public GraphQL surface being stable. If Railway changes its response shape, tests will fail loudly — that's the point.
- The `railway_deployment_trigger` test in Tier 4 depends on the workspace having the Railway GitHub app connected to at least the public target repo. If not, that test will fail; the rest are unaffected.
- Mock-based unit tests using `resource.UnitTest` currently trip on an OpenTofu dependency lock-file issue when invoked under `go test`; this is a test-harness quirk, not a runtime bug. The retry-classification helpers ARE covered by plain-Go unit tests exercised in Tier 0.
