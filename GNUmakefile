default: testacc

# OpenTofu test configuration — required for unit tests that use resource.UnitTest()
# Without these, OpenTofu rejects the legacy "-" provider namespace.
# See: https://github.com/opentofu/opentofu/issues/977
TOFU_TEST_ENV = TF_ACC_TERRAFORM_PATH=$(shell which tofu) \
	TF_ACC_PROVIDER_NAMESPACE=hashicorp \
	TF_ACC_PROVIDER_HOST=registry.opentofu.org

# Run acceptance tests (requires RAILWAY_TOKEN env var)
.PHONY: testacc
testacc:
	$(TOFU_TEST_ENV) TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Run unit tests only (mock-based, no Railway token needed)
.PHONY: test
test:
	$(TOFU_TEST_ENV) go test ./internal/provider/ -v -timeout 5m

download-schema:
	npx get-graphql-schema https://backboard.railway.app/graphql/v2 > schema.graphql
