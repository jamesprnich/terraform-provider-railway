package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// v0.11.0 bumped the railway_service schema Version 1 → 2 as part of the
// per-environment shell redesign — `source_*`, `cron_schedule`, `regions`, and
// several other attributes moved off `railway_service` onto
// `railway_service_instance`. That transformation is not automatable (the
// same v1 railway_service state maps to multiple v2 resources across
// environments), and the v0.11.0 CHANGELOG explicitly directs users to
// `terraform state rm` any pre-v0.11.0 `railway_service` resources and
// re-import them under the new shape.
//
// v0.11.0 shipped without an UpgradeState registration, so users who did NOT
// follow that CHANGELOG migration hit the framework's cryptic diagnostic:
//
//	Unable to Upgrade Resource State — This resource was implemented without
//	an UpgradeState() method, however Terraform was expecting an
//	implementation for version 1 upgrade.
//
// v0.11.3 registers a rejecting upgrader that surfaces a clear migration
// instruction instead. The upgrader never returns state — it always emits an
// error diagnostic — so users are forced to take the documented manual
// migration path.
//
// PriorSchema is intentionally omitted (nil). Per
// `terraform-plugin-framework/internal/fwserver/server_upgraderesourcestate.go`
// (line 170 in v1.19.0), the framework only attempts to unmarshal the raw
// state against PriorSchema when it is non-nil — and if that unmarshal fails
// (which it would against an incomplete PriorSchema), the framework surfaces
// "Unable to Read Previously Saved State" and never calls our StateUpgrader,
// meaning our helpful rejecting message never fires. Leaving PriorSchema nil
// skips the unmarshal entirely; the framework passes req.RawState through
// and calls our upgrader as intended.
func (r *ServiceResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		1: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				resp.Diagnostics.AddError(
					"Manual migration required for railway_service",
					"State for railway_service was written by the provider before v0.11.0, when the resource "+
						"carried source, build, and deploy configuration inline. v0.11.0 redesigned railway_service "+
						"as a per-environment shell and moved every one of those attributes onto "+
						"railway_service_instance. That transformation is not automatable — the same v1 state maps "+
						"to multiple v2 resources across environments — so no upgrader can perform it.\n\n"+
						"Follow the v0.11.0 CHANGELOG migration: remove affected railway_service resources from "+
						"state (`tofu state rm railway_service.<name>`), update your configuration to match the "+
						"v0.11.0 shape (environment-scoped services + railway_service_instance for build/deploy "+
						"config), then re-import the underlying Railway service.",
				)
			},
		},
	}
}
