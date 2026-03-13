#!/usr/bin/env bash
# Compares the local schema.graphql hash to the recorded value in schema_version.go.
# Usage: ./scripts/check-schema.sh (run from repository root)
set -euo pipefail

SCHEMA_FILE="schema.graphql"
VERSION_FILE="internal/provider/schema_version.go"

RECORDED_HASH=$(grep 'SchemaVersion' "$VERSION_FILE" | head -1 | sed 's/.*"\(.*\)".*/\1/')

if [ -z "$RECORDED_HASH" ]; then
  echo "ERROR: Could not extract SchemaVersion from $VERSION_FILE"
  exit 2
fi

if [ ! -f "$SCHEMA_FILE" ]; then
  echo "ERROR: $SCHEMA_FILE not found. Run from the repository root."
  exit 2
fi

CURRENT_HASH=$(sha256sum "$SCHEMA_FILE" | awk '{print $1}')

if [ "$CURRENT_HASH" = "$RECORDED_HASH" ]; then
  echo "OK: Schema hash matches recorded value ($RECORDED_HASH)"
  exit 0
else
  echo "DRIFT DETECTED"
  echo "  Recorded: $RECORDED_HASH"
  echo "  Current:  $CURRENT_HASH"
  echo ""
  echo "Update SchemaVersion in $VERSION_FILE and re-audit for API changes."
  exit 1
fi
