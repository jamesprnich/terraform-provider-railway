## Requirements

- [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.9 (or [Terraform](https://www.terraform.io/downloads.html) >= 1.0)
- [Go](https://golang.org/doc/install) >= 1.25

## Building The Provider

1. Clone the repository
2. Build the provider:

```shell
go build -o ~/go/bin/terraform-provider-railway .
```

## GraphQL Client Regeneration

If you modify any `.graphql` files in `internal/provider/`, regenerate the client:

```shell
go run github.com/Khan/genqlient
```

> **Warning:** Do NOT use `go generate` — it also runs `terraform fmt` and `tfplugindocs`, which require a Terraform binary and may overwrite hand-edited documentation.

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Testing

Unit tests use mock HTTP servers and do not require a Railway account:

```shell
make test
```

Acceptance tests create real resources and require a `RAILWAY_TOKEN` environment variable:

```shell
make testacc
```

Both targets automatically set the OpenTofu compatibility environment variables. If running `go test` directly, you must set them yourself:

```shell
TF_ACC_TERRAFORM_PATH="$(which tofu)" \
TF_ACC_PROVIDER_NAMESPACE="hashicorp" \
TF_ACC_PROVIDER_HOST="registry.opentofu.org" \
go test ./internal/provider/ -v
```

## Schema Versioning

The provider tracks the Railway GraphQL schema version. To check for schema drift:

```shell
./scripts/check-schema.sh
```

## Documentation

Registry docs live in `docs/resources/` and `docs/data-sources/`. Examples live in `examples/resources/` and `examples/data-sources/`.

When adding a new resource, create:
- `docs/resources/<name>.md` — registry documentation page
- `examples/resources/railway_<name>/resource.tf` — example configuration
- `examples/resources/railway_<name>/import.sh` — import command example
