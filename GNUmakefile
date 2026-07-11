# =============================================================================
# terraform-provider-railway — developer targets
# =============================================================================
# The one you'll use daily:
#
#   make check           full local pipeline: lint + test + build
#   make lint            golangci-lint (same set CI runs)
#   make lint-fix        golangci-lint --fix
#   make test            unit tests, mock-only, no live Railway
#   make testacc         acceptance tests (requires RAILWAY_TOKEN)
#   make build           compile the provider
#   make docs            regenerate registry docs (preserves MkDocs content)
#   make generate        regenerate genqlient GraphQL client
#   make security        run govulncheck against Go stdlib + deps
#   make download-schema pull latest schema.graphql from Railway
# =============================================================================

GOLANGCI_LINT_VERSION := v2.4.0

# OpenTofu test configuration — required for unit tests that use resource.UnitTest().
# Without these, OpenTofu rejects the legacy "-" provider namespace.
# See: https://github.com/opentofu/opentofu/issues/977
TOFU_TEST_ENV = TF_ACC_TERRAFORM_PATH=$(shell which tofu) \
	TF_ACC_PROVIDER_NAMESPACE=jamesprnich \
	TF_ACC_PROVIDER_HOST=registry.opentofu.org

.PHONY: default check lint lint-fix test testacc build docs generate security tools download-schema clean

default: check

check: lint test build
	@echo "✓ check: lint + test + build all green"

## Ensure golangci-lint is installed at the pinned version.
tools:
	@if ! command -v golangci-lint >/dev/null 2>&1 || \
	   ! golangci-lint --version 2>/dev/null | grep -q "$(GOLANGCI_LINT_VERSION:v%=%)"; then \
	    echo "→ installing golangci-lint $(GOLANGCI_LINT_VERSION)"; \
	    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

lint: tools
	golangci-lint run --timeout=5m ./...

lint-fix: tools
	golangci-lint run --fix --timeout=5m ./...

# Run unit tests only (mock-based, no Railway token needed)
test:
	$(TOFU_TEST_ENV) go test ./internal/provider/ -v -timeout 5m

# Run acceptance tests (requires RAILWAY_TOKEN env var)
testacc:
	$(TOFU_TEST_ENV) TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

build:
	go build ./...

docs:
	bash scripts/regen-docs.sh

generate:
	cd internal/provider && go run github.com/Khan/genqlient

security:
	@command -v govulncheck >/dev/null 2>&1 || \
	    go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

download-schema:
	npx get-graphql-schema https://backboard.railway.app/graphql/v2 > schema.graphql

clean:
	rm -rf terraform-provider-railway coverage.out site
