package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestUpgradePreDeployCommandV1ToV2 exercises the pure transformation. This
// is the HashiCorp-pattern per-upgrader test — build a prior-shape value by
// hand, call the upgrader, assert the new-shape value. Covers every case the
// upgrader's contract documents:
//
//   - null list             → null string
//   - unknown list          → null string
//   - empty list            → null string
//   - single-element list   → string with that element
//   - multi-element list    → strings.Join(elts, " && ")
//
// The multi-element case is defensive — the v0.11.1/v0.11.2 provider never
// produced multi-element state (Railway rejects it and the v0.11.2 provider
// serialises exactly one), but hand-edited state files or a future schema
// relaxation could carry >1 element. Joining is the reporter's recommended
// lossless collapse.
func TestUpgradePreDeployCommandV1ToV2(t *testing.T) {
	ctx := context.Background()

	strList := func(vals ...string) types.List {
		if vals == nil {
			return types.ListNull(types.StringType)
		}
		elems := make([]attr.Value, len(vals))
		for i, v := range vals {
			elems[i] = types.StringValue(v)
		}
		l, diags := types.ListValue(types.StringType, elems)
		if diags.HasError() {
			t.Fatalf("build test list: %+v", diags)
		}
		return l
	}

	tests := []struct {
		name string
		in   types.List
		want types.String
	}{
		{
			name: "null list becomes null string — the only shape v0.11.1 could produce, so this is the load-bearing case",
			in:   types.ListNull(types.StringType),
			want: types.StringNull(),
		},
		{
			name: "unknown list becomes null string",
			in:   types.ListUnknown(types.StringType),
			want: types.StringNull(),
		},
		{
			name: "empty list becomes null string",
			in:   strList(),
			want: types.StringNull(),
		},
		{
			name: "single element carries through unchanged",
			in:   strList("python manage.py migrate"),
			want: types.StringValue("python manage.py migrate"),
		},
		{
			name: "two elements join with ' && '",
			in:   strList("python manage.py migrate", "python manage.py collectstatic --noinput"),
			want: types.StringValue("python manage.py migrate && python manage.py collectstatic --noinput"),
		},
		{
			name: "three elements join with ' && ' in order",
			in:   strList("a", "b", "c"),
			want: types.StringValue("a && b && c"),
		},
		{
			name: "element containing shell metacharacters is preserved verbatim",
			in:   strList("echo 'hello world' | tee /tmp/x"),
			want: types.StringValue("echo 'hello world' | tee /tmp/x"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := upgradePreDeployCommandV1ToV2(ctx, tc.in)
			if diags.HasError() {
				t.Fatalf("upgradePreDeployCommandV1ToV2 unexpected diagnostics: %+v", diags)
			}
			if !got.Equal(tc.want) {
				t.Errorf("upgradePreDeployCommandV1ToV2\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
