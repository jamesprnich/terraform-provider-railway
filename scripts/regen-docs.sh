#!/usr/bin/env bash
# =============================================================================
# regen-docs.sh — regenerate registry docs without wiping MkDocs content
# =============================================================================
# tfplugindocs generate rebuilds the entire `docs/` directory from
# `templates/`, wiping anything under `docs/guides/`, `docs/overrides/`, and
# `docs/home.md`. Those files are hand-authored MkDocs content and must
# survive regeneration. This script snapshots them, runs tfplugindocs, and
# restores them.
#
# It also passes `--provider-name railway` so filenames match the shipped
# convention (`docs/resources/service.md`, not `docs/resources/railway_service.md`).
#
# Run from repo root: bash scripts/regen-docs.sh
# =============================================================================
set -euo pipefail

PRESERVE=(
    "docs/home.md"
    "docs/guides"
    "docs/overrides"
)

SNAPSHOT=$(mktemp -d)
trap 'rm -rf "$SNAPSHOT"' EXIT

echo "→ snapshotting hand-authored MkDocs content..."
for path in "${PRESERVE[@]}"; do
    if [[ -e "$path" ]]; then
        mkdir -p "$SNAPSHOT/$(dirname "$path")"
        cp -r "$path" "$SNAPSHOT/$path"
        echo "  ✓ $path"
    fi
done

echo ""
echo "→ running tfplugindocs generate..."
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate \
    --provider-name railway 2>&1 | tail -5

echo ""
echo "→ restoring hand-authored MkDocs content..."
for path in "${PRESERVE[@]}"; do
    if [[ -e "$SNAPSHOT/$path" ]]; then
        rm -rf "$path"
        mkdir -p "$(dirname "$path")"
        cp -r "$SNAPSHOT/$path" "$path"
        echo "  ✓ $path"
    fi
done

echo ""
echo "✓ docs regenerated"
echo ""
echo "Verify with:"
echo "  ls docs/resources/    # should list all resource docs with short names"
echo "  ls docs/data-sources/ # should list data source docs"
echo "  ls docs/guides/       # should include hand-authored guides"
