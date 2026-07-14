package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// v0.11.0 bumped the railway_environment schema Version 1 → 2 as part of the
// strict env-scoping design — `source_environment_id` became a first-class
// required attribute under strict mode. Migrating v1 state to v2 is not
// automatable in the general case: a legacy railway_environment with no
// source (a "non-fork" env) cannot become a fork after the fact, and strict
// mode rejects it outright at plan time. The v0.11.0 CHANGELOG directs users
// to review their environments and either opt into permissive mode
// (`strict_env_scoping = false`) or `terraform state rm` and recreate any
// non-fork environments as forks.
//
// v0.11.0 shipped without an UpgradeState registration; the framework's
// cryptic "was expecting an implementation for version 1 upgrade" diagnostic
// was the only error users saw. v0.11.3 registers a rejecting upgrader that
// surfaces the actual migration instruction. Same pattern (and rationale for
// nil PriorSchema) as resource_service_upgrade.go.
func (r *EnvironmentResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		1: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				resp.Diagnostics.AddError(
					"Manual migration required for railway_environment",
					"State for railway_environment was written by the provider before v0.11.0. v0.11.0 introduced "+
						"strict env-scoping and made `source_environment_id` a first-class attribute; under strict "+
						"mode (provider default) every environment must be a fork.\n\n"+
						"Two migration paths:\n\n"+
						"  1. Recommended: opt into strict mode. Remove any non-fork railway_environment resources "+
						"from state (`tofu state rm railway_environment.<name>`), update your configuration to set "+
						"`source_environment_id`, then re-import.\n\n"+
						"  2. Opt out: set `strict_env_scoping = false` on the provider block and re-run — the v1 "+
						"state will be accepted as-is under permissive mode.\n\n"+
						"See the v0.11.0 CHANGELOG for the design rationale and full migration guidance.",
				)
			},
		},
	}
}
