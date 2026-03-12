#!/usr/bin/env bash
# =============================================================================
# Deploy Environment Layer
# =============================================================================
# Creates/updates a Railway environment with service instances, Postgres
# volume, variables, and a public domain. Uses OpenTofu workspaces for
# per-environment state isolation.
#
# Prerequisites:
#   - deploy-services.sh has been run (project + services exist)
#
# Usage:
#   export RAILWAY_TOKEN="your-account-token"
#   ./deploy-environment.sh dev                # plan + apply "dev"
#   ./deploy-environment.sh qa                 # plan + apply "qa"
#   ./deploy-environment.sh dev plan           # plan only
#   ./deploy-environment.sh dev destroy        # tear down "dev" environment
#
# Required:
#   export RAILWAY_TOKEN="your-account-token"
#
# Optional (prompted if not set):
#   export TF_VAR_postgres_password="your-password"
#   export TF_VAR_app_repo="owner/railway-terraform-provider"
#   export TF_VAR_project_name="my-app"        # default: test-app
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENVS_DIR="$SCRIPT_DIR/environments"
ENV_NAME="${1:-}"
ACTION="${2:-apply}"

# --- Validate ---

if [[ -z "$ENV_NAME" ]]; then
    echo "Usage: $0 <environment> [plan|apply|destroy]"
    echo ""
    echo "Examples:"
    echo "  $0 dev          # create/update dev environment"
    echo "  $0 qa           # create/update qa environment"
    echo "  $0 prd          # create/update prd environment"
    echo "  $0 dev destroy  # tear down dev environment"
    exit 1
fi

if [[ -z "${RAILWAY_TOKEN:-}" ]]; then
    read -rsp "Railway account token: " RAILWAY_TOKEN
    echo ""
    export RAILWAY_TOKEN
fi

if [[ "$ACTION" != "plan" && "$ACTION" != "apply" && "$ACTION" != "destroy" ]]; then
    echo "Usage: $0 <environment> [plan|apply|destroy]"
    exit 1
fi

# --- Prompt for required variables if not set ---

if [[ -z "${TF_VAR_postgres_password:-}" ]]; then
    read -rsp "Postgres password: " TF_VAR_postgres_password
    echo ""
    export TF_VAR_postgres_password
fi

if [[ -z "${TF_VAR_app_repo:-}" ]]; then
    read -rp "App repo (e.g. owner/railway-terraform-provider): " TF_VAR_app_repo
    export TF_VAR_app_repo
fi

# --- Init + workspace ---

echo "=== Environment Layer ==="
echo "Environment: $ENV_NAME"
echo "Action: $ACTION"
echo "Project: ${TF_VAR_project_name:-test-app}"
echo "App repo: $TF_VAR_app_repo"
echo ""

cd "$ENVS_DIR"
tofu init -input=false
tofu workspace select -or-create "$ENV_NAME"

# --- Execute ---

if [[ "$ACTION" == "destroy" ]]; then
    echo ""
    echo "WARNING: This will destroy the '$ENV_NAME' environment and all its resources."
    read -rp "Type 'yes' to confirm: " confirm
    if [[ "$confirm" != "yes" ]]; then
        echo "Aborted."
        exit 1
    fi
    tofu destroy -auto-approve -parallelism=5
else
    tofu plan -out=environment.tfplan

    if [[ "$ACTION" == "apply" ]]; then
        echo ""
        tofu apply -parallelism=5 environment.tfplan
        echo ""
        echo "=== $ENV_NAME Deployed ==="
        echo "Environment ID: $(tofu output -raw environment_id)"
        echo "App URL: $(tofu output -raw app_url)"
    fi
fi
