#!/usr/bin/env bash
# Comprehensive manual test regime for the Railway Terraform provider.
# Not for CI/CD — a human runs this against a real Railway workspace before
# cutting a release or after material CRUD changes. See ./README.md.
#
# Usage:
#   RAILWAY_TOKEN=... RAILWAY_TEST_WORKSPACE_ID=... ./run.sh
#
# Optional env:
#   PROVIDER_BINARY_DIR   default: $(go env GOPATH)/bin
#   TEST_PREFIX           default: AAA-provctest- (used only by check-workspace.sh)
#   LOG_DIR               default: ./logs-<timestamp>
#   TIERS                 default: "0 1 2 3 4"  (space-separated tiers to run)
#   COOLDOWN_SECONDS      default: 30  (pause between tests / tiers)
#
# All tests run sequentially. Each test creates its own throwaway Railway
# project prefixed AAA-provctest-*, and destroys it at the end. Post-flight
# checks the workspace for any lingering AAA-provctest-* project and reports.
set -uo pipefail

# -------- required env --------
: "${RAILWAY_TOKEN:?RAILWAY_TOKEN must be set — Railway API token with project-create permission}"
: "${RAILWAY_TEST_WORKSPACE_ID:?RAILWAY_TEST_WORKSPACE_ID must be set — workspace ID (needed by notification_rule)}"

# -------- optional env --------
PROVIDER_BINARY_DIR="${PROVIDER_BINARY_DIR:-$(go env GOPATH)/bin}"
COOLDOWN_SECONDS="${COOLDOWN_SECONDS:-30}"
TIERS="${TIERS:-0 1 2 3 4}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_DIR="$SCRIPT_DIR/tests"
LOG_DIR="${LOG_DIR:-$SCRIPT_DIR/logs-$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$LOG_DIR"

# -------- one tofurc for the whole run --------
TOFURC="$LOG_DIR/tofurc"
sed "s|@@PROVIDER_BINARY_DIR@@|$PROVIDER_BINARY_DIR|" "$SCRIPT_DIR/tofurc.template" > "$TOFURC"
export TF_CLI_CONFIG_FILE="$TOFURC"

# -------- results --------
declare -a PASSED FAILED SKIPPED
run_test() {
  local name="$1" dir="$2" ; shift 2
  echo ""
  echo "===================================================================="
  echo "  $name"
  echo "===================================================================="
  cd "$dir"
  rm -f terraform.tfstate terraform.tfstate.backup .terraform.lock.hcl
  rm -rf .terraform
  local start=$(date +%s)
  if tofu apply -auto-approve "$@" 2>&1 | tee "$LOG_DIR/${name}.apply.log" > /dev/null ; then
    local ae=0
  else
    local ae="${PIPESTATUS[0]}"
  fi
  local ad=$(($(date +%s) - start))
  echo "  APPLY exit=$ae dur=${ad}s"

  start=$(date +%s)
  tofu destroy -auto-approve "$@" 2>&1 | tee "$LOG_DIR/${name}.destroy.log" > /dev/null
  local de="${PIPESTATUS[0]}"
  local dd=$(($(date +%s) - start))
  echo "  DESTROY exit=$de dur=${dd}s"

  if [ "$de" -ne 0 ]; then
    echo "  (retrying destroy after ${COOLDOWN_SECONDS}s)"
    sleep "$COOLDOWN_SECONDS"
    tofu destroy -auto-approve "$@" 2>&1 | tee "$LOG_DIR/${name}.destroy-retry.log" > /dev/null
    de="${PIPESTATUS[0]}"
    echo "  RETRY_DESTROY exit=$de"
  fi

  if [ "$ae" -eq 0 ] && [ "$de" -eq 0 ]; then
    PASSED+=("$name")
  else
    FAILED+=("$name (apply=$ae destroy=$de)")
  fi
}

run_plan_only_expect_fail() {
  local name="$1" dir="$2" ; shift 2
  local pattern="$1" ; shift
  echo ""
  echo "===================================================================="
  echo "  $name (plan-only, expects NON-zero + pattern match)"
  echo "===================================================================="
  cd "$dir"
  rm -f terraform.tfstate*
  rm -rf .terraform
  tofu plan "$@" 2>&1 | tee "$LOG_DIR/${name}.plan.log" > /dev/null
  local pe="${PIPESTATUS[0]}"
  echo "  PLAN exit=$pe (want NON-zero)"
  if [ "$pe" -eq 0 ]; then
    FAILED+=("$name — plan should have failed but exited 0")
    return
  fi
  if grep -qE "$pattern" "$LOG_DIR/${name}.plan.log"; then
    PASSED+=("$name")
  else
    FAILED+=("$name — plan failed but expected pattern '$pattern' not found")
  fi
}

# ====================================================================
# TIER 0 — offline
# ====================================================================
if [[ " $TIERS " == *" 0 "* ]]; then
  echo ""
  echo "############### TIER 0 — offline ###############"
  cd "$(dirname "$SCRIPT_DIR")/.."   # repo root
  if go build ./... 2>&1 | tee "$LOG_DIR/tier0.build.log" ; then
    echo "  go build: PASS"
  else
    FAILED+=("tier0/build")
    exit 2
  fi
  if go vet ./... 2>&1 | tee "$LOG_DIR/tier0.vet.log" ; then
    echo "  go vet: PASS"
  else
    FAILED+=("tier0/vet")
  fi
  if go test -count=1 -run '^TestIsNotFound_|^TestRetryReadAfterCreateContext_|^TestIsRedeploy|^TestRetryRedeployContext' ./internal/provider/ 2>&1 | tee "$LOG_DIR/tier0.tests.log" ; then
    echo "  helpers unit tests: PASS"
  else
    FAILED+=("tier0/unit-tests")
  fi
  PASSED+=("tier0/build+vet+unit-tests")

  echo ""
  echo "  Pre-flight workspace check (no leftover AAA-provctest-* projects)..."
  if "$SCRIPT_DIR/check-workspace.sh" 2>&1 | tee "$LOG_DIR/tier0.workspace-check.log" ; then
    echo "  workspace: CLEAN"
  else
    echo "  workspace has leftover projects — refusing to run further tiers"
    echo "  (destroy them manually via the Railway dashboard, then re-run)"
    exit 3
  fi
fi

# ====================================================================
# TIER 1 — cheap live
# ====================================================================
if [[ " $TIERS " == *" 1 "* ]]; then
  echo ""
  echo "############### TIER 1 — project, env, data sources ###############"
  sleep "$COOLDOWN_SECONDS"
  run_test "t1_1_project_crud"       "$TESTS_DIR/t1_1_project_crud"
  sleep "$COOLDOWN_SECONDS"
  run_test "t1_2_env_fork_nonfork"   "$TESTS_DIR/t1_2_env_fork_nonfork"
  sleep "$COOLDOWN_SECONDS"
  run_test "t1_3_data_sources"       "$TESTS_DIR/t1_3_data_sources"
  sleep "$COOLDOWN_SECONDS"
  run_plan_only_expect_fail "t1_4_strict_plan_reject" "$TESTS_DIR/t1_4_strict_plan_reject" \
    'environment_id is required under strict env-scoping|source_environment_id is required under strict env-scoping'
fi

# ====================================================================
# TIER 2 — non-compute resources
# ====================================================================
if [[ " $TIERS " == *" 2 "* ]]; then
  echo ""
  echo "############### TIER 2 — resources, no compute ###############"
  sleep "$COOLDOWN_SECONDS"
  run_test "t2_all_resources_no_compute" \
    "$TESTS_DIR/t2_all_resources_no_compute" \
    -var="workspace_id=$RAILWAY_TEST_WORKSPACE_ID"
fi

# ====================================================================
# TIER 3 — compute deploys
# ====================================================================
if [[ " $TIERS " == *" 3 "* ]]; then
  echo ""
  echo "############### TIER 3 — compute deploys ###############"
  sleep "$COOLDOWN_SECONDS"
  run_test "t3_1_fork_topology"          "$TESTS_DIR/t3_1_fork_topology"
  sleep "$COOLDOWN_SECONDS"
  run_test "t3_2_e2e_both_volume_paths"  "$TESTS_DIR/t3_2_e2e_both_volume_paths"
fi

# ====================================================================
# TIER 4 — stress + edge
# ====================================================================
if [[ " $TIERS " == *" 4 "* ]]; then
  echo ""
  echo "############### TIER 4 — stress + edge ###############"
  # Flake: run 5 iterations of the same shape with distinct suffixes.
  for i in 1 2 3 4 5; do
    sleep "$COOLDOWN_SECONDS"
    run_test "t4_1_flake_$i" "$TESTS_DIR/t4_1_flake" -var="suffix=$i"
  done
  sleep "$COOLDOWN_SECONDS"
  run_test "t4_2_rename_create" "$TESTS_DIR/t4_2_rename"
  # T4.2 second phase (rename): re-apply with renamed vars, then destroy.
  # Handled as a separate run_test-like block below since it needs a
  # two-apply-then-destroy sequence.
  sleep "$COOLDOWN_SECONDS"
  echo ""
  echo "===================================================================="
  echo "  t4_2_rename_lifecycle (two applies + destroy)"
  echo "===================================================================="
  cd "$TESTS_DIR/t4_2_rename"
  rm -f terraform.tfstate*
  tofu apply -auto-approve                                        > "$LOG_DIR/t4_2_rename.a1.log" 2>&1 ; a1=$?
  tofu apply -auto-approve -var="svc_name=dev-svc-renamed" \
                           -var="env_name=dev-renamed" \
                           -var="vol_name=new-vol"                > "$LOG_DIR/t4_2_rename.a2.log" 2>&1 ; a2=$?
  tofu destroy -auto-approve -var="svc_name=dev-svc-renamed" \
                             -var="env_name=dev-renamed" \
                             -var="vol_name=new-vol"              > "$LOG_DIR/t4_2_rename.destroy.log" 2>&1 ; d=$?
  echo "  apply1=$a1 apply2=$a2 destroy=$d"
  if [ $a1 -eq 0 ] && [ $a2 -eq 0 ] && [ $d -eq 0 ]; then
    PASSED+=("t4_2_rename_lifecycle")
  else
    FAILED+=("t4_2_rename_lifecycle (a1=$a1 a2=$a2 d=$d)")
  fi

  sleep "$COOLDOWN_SECONDS"
  # T4.3 collision — apply is EXPECTED to fail with the real "already exists"
  # error. Destroy MUST still succeed.
  echo ""
  echo "===================================================================="
  echo "  t4_3_collision (apply expected to fail, destroy expected to succeed)"
  echo "===================================================================="
  cd "$TESTS_DIR/t4_3_collision"
  rm -f terraform.tfstate*
  tofu apply -auto-approve  > "$LOG_DIR/t4_3_collision.apply.log" 2>&1 ; a=$?
  tofu destroy -auto-approve > "$LOG_DIR/t4_3_collision.destroy.log" 2>&1 ; d=$?
  echo "  apply=$a (want NON-zero) destroy=$d (want 0)"
  if [ $a -ne 0 ] && [ $d -eq 0 ] && grep -q "already exists in this project" "$LOG_DIR/t4_3_collision.apply.log" ; then
    PASSED+=("t4_3_collision")
  else
    FAILED+=("t4_3_collision (a=$a d=$d)")
  fi

  # T4.4 rapid cycles — 2 iterations, distinct suffixes.
  for i in 1 2; do
    sleep "$COOLDOWN_SECONDS"
    run_test "t4_4_rapid_cycle_$i" "$TESTS_DIR/t4_4_rapid_cycle" -var="suffix=$i"
  done

  sleep "$COOLDOWN_SECONDS"
  run_test "t4_5_deployment_trigger" "$TESTS_DIR/t4_5_deployment_trigger"
fi

# ====================================================================
# Post-flight
# ====================================================================
echo ""
echo "############### POST-FLIGHT ###############"
if "$SCRIPT_DIR/check-workspace.sh" 2>&1 | tee "$LOG_DIR/postflight.workspace-check.log" ; then
  echo "workspace: CLEAN — no lingering AAA-provctest-* projects"
else
  echo "workspace has orphans — inspect logs and consider manual destroy in the Railway dashboard"
fi

# ====================================================================
# Summary
# ====================================================================
echo ""
echo "===================================================================="
echo "  RESULT SUMMARY (logs: $LOG_DIR)"
echo "===================================================================="
echo "PASSED (${#PASSED[@]}):"
printf '  ✔ %s\n' "${PASSED[@]}" 2>/dev/null || echo "  (none)"
if [ "${#FAILED[@]}" -gt 0 ]; then
  echo ""
  echo "FAILED (${#FAILED[@]}):"
  printf '  ✗ %s\n' "${FAILED[@]}"
  exit 1
fi
echo ""
echo "All tests passed."
