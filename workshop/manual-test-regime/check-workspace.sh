#!/usr/bin/env bash
# Read-only workspace hygiene check.
# Lists any project whose name starts with the test prefix ($TEST_PREFIX or
# default "AAA-provctest-"). Exit 0 if none, 1 if any exist.
set -uo pipefail

: "${RAILWAY_TOKEN:?RAILWAY_TOKEN must be set — Railway API token}"
PREFIX="${TEST_PREFIX:-AAA-provctest-}"

response=$(curl -sf -X POST https://backboard.railway.com/graphql/v2 \
  -H "Authorization: Bearer $RAILWAY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"{ projects { edges { node { id name } } } }"}')

if [ $? -ne 0 ] || [ -z "$response" ]; then
  echo "ERROR: Railway API call failed. Check RAILWAY_TOKEN and network." >&2
  exit 2
fi

matches=$(echo "$response" | jq -r --arg pfx "$PREFIX" \
  '.data.projects.edges[] | select(.node.name | startswith($pfx)) | "  \(.node.id)  \(.node.name)"')

if [ -z "$matches" ]; then
  echo "OK: workspace has no projects matching prefix '$PREFIX'"
  exit 0
fi

echo "FAIL: workspace has projects matching prefix '$PREFIX':"
echo "$matches"
exit 1
