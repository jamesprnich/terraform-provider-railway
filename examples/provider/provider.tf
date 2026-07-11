# Railway provider configuration.
#
# strict_env_scoping defaults to `true` in v0.11.0+ — services and additional
# environments must be explicitly env-scoped or the plan fails. Set to false
# to opt out (pre-v0.11.0 semantics: `serviceCreate` and `environmentCreate`
# without env scoping create resources across every non-fork environment).

provider "railway" {
  # token = "..."                # or set via RAILWAY_TOKEN environment variable
  # strict_env_scoping = false   # opt out of strict env-scoping enforcement
}
