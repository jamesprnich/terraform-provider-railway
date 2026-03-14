package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// useStateForUnknownIfNonNull returns a plan modifier that copies the prior
// state value into the plan when the plan value is unknown AND the prior state
// value is non-null. This is needed for computed attributes inside optional
// nested blocks (e.g., volume.id inside a service). The built-in
// UseStateForUnknown copies null from the prior state when the parent block
// transitions from null to non-null, which causes "inconsistent result after
// apply" errors.

func useStringStateForUnknownIfNonNull() planmodifier.String {
	return stringStateForUnknownIfNonNull{}
}

type stringStateForUnknownIfNonNull struct{}

func (m stringStateForUnknownIfNonNull) Description(_ context.Context) string {
	return "Use prior state for unknown values, but only when prior state is non-null."
}

func (m stringStateForUnknownIfNonNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m stringStateForUnknownIfNonNull) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

func useFloat64StateForUnknownIfNonNull() planmodifier.Float64 {
	return float64StateForUnknownIfNonNull{}
}

type float64StateForUnknownIfNonNull struct{}

func (m float64StateForUnknownIfNonNull) Description(_ context.Context) string {
	return "Use prior state for unknown values, but only when prior state is non-null."
}

func (m float64StateForUnknownIfNonNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m float64StateForUnknownIfNonNull) PlanModifyFloat64(_ context.Context, req planmodifier.Float64Request, resp *planmodifier.Float64Response) {
	if !req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}
