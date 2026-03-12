#!/usr/bin/env bash
# =============================================================================
# Deploy Services Layer
# =============================================================================
# Creates the Railway project and empty services. Run this once before
# deploying any environments.
#
# Usage:
#   export RAILWAY_TOKEN="your-account-token"
#   ./deploy-services.sh              # plan + apply
#   ./deploy-services.sh plan         # plan only
#   ./deploy-services.sh destroy      # tear down services
#
# Optional:
#   export TF_VAR_project_name="my-app"   # default: test-app
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICES_DIR="$SCRIPT_DIR/services"
ACTION="${1:-apply}"

# --- Validate ---

if [[ -z "${RAILWAY_TOKEN:-}" ]]; then
    read -rsp "Railway account token: " RAILWAY_TOKEN
    echo ""
    export RAILWAY_TOKEN
fi

if [[ "$ACTION" != "plan" && "$ACTION" != "apply" && "$ACTION" != "destroy" ]]; then
    echo "Usage: $0 [plan|apply|destroy]"
    exit 1
fi

# --- Init ---

echo "=== Services Layer ==="
echo "Action: $ACTION"
echo "Project: ${TF_VAR_project_name:-test-app}"
echo ""

cd "$SERVICES_DIR"
tofu init -input=false

# --- Execute ---

if [[ "$ACTION" == "destroy" ]]; then
    echo ""
    echo "WARNING: This will destroy the Railway project and all services."
    read -rp "Type 'yes' to confirm: " confirm
    if [[ "$confirm" != "yes" ]]; then
        echo "Aborted."
        exit 1
    fi
    tofu destroy -auto-approve
else
    tofu plan -out=services.tfplan

    if [[ "$ACTION" == "apply" ]]; then
        echo ""
        tofu apply services.tfplan
        echo ""
        echo "=== Services Created ==="
        echo "Project ID: $(tofu output -raw project_id)"
        echo "Project Name: $(tofu output -raw project_name)"
        echo ""
        echo "Next: deploy an environment:"
        echo "  ./deploy-environment.sh dev"
    fi
fi
