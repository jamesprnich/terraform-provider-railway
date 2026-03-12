#!/usr/bin/env bash
# Fetches the current Railway GraphQL schema via introspection and compares
# its SHA256 hash to the recorded value in schema_version.go.
set -euo pipefail

SCHEMA_FILE="schema.graphql"
RECORDED_HASH="214c96676337fddcad1b673bed74989ca4198a573dd7f683c9360b4529d65b8e"

if [ ! -f "$SCHEMA_FILE" ]; then
  echo "ERROR: $SCHEMA_FILE not found. Run from the repository root."
  exit 1
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
  echo "Update SchemaVersion in internal/provider/schema_version.go and re-audit for API changes."
  exit 1
fi
