package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestSchemaVersionsHaveUpgraders enforces the framework invariant that
// shipped v0.11.2 broken: whenever a resource's schema Version is bumped,
// the resource must register an UpgradeState with an entry for every prior
// version, or every existing user's next plan/apply fails with "Unable to
// Upgrade Resource State — was expecting an implementation for version N
// upgrade."
//
// The Plugin Framework does not perform this check at registration time —
// it only surfaces the missing upgrader at the first refresh RPC after a
// version mismatch is observed, i.e. against a real user's state. That is
// how v0.11.2 slipped through mock unit tests, live acceptance, and code
// review: none of those exercised state carrying a prior-version stamp.
//
// This test walks every resource the provider registers and asserts that if
// current Version ≥ 2, the resource implements ResourceWithUpgradeState AND
// its UpgradeState returns a map with entries for every source version in
// [1..Version-1]. (Version 1 resources are intentionally out of scope: at
// runtime we cannot distinguish "born at Version 1 — no prior state exists"
// from "bumped from Version 0 — needs an upgrader" without git-history
// context. The invariant is scoped to bumps ≥ 2 because that is the class
// of bug v0.11.2 shipped and the class actively caught by this test.)
//
// If this test had existed in v0.11.2, it would have failed on
// ServiceInstanceResource going from Version 1 to Version 2 without an
// UpgradeState, blocking the release.
func TestSchemaVersionsHaveUpgraders(t *testing.T) {
	ctx := context.Background()
	provider := New("test")()

	metaResp := struct {
		TypeName string
	}{}
	_ = metaResp

	// Provider.Resources returns constructors; instantiate each and read
	// its Schema + Metadata.
	prov, ok := provider.(interface {
		Resources(context.Context) []func() resource.Resource
	})
	if !ok {
		t.Fatalf("provider does not implement Resources(ctx) []func() resource.Resource")
	}

	for _, factory := range prov.Resources(ctx) {
		res := factory()

		// Type name for readable failure messages.
		mdReq := resource.MetadataRequest{ProviderTypeName: "railway"}
		mdResp := resource.MetadataResponse{}
		res.Metadata(ctx, mdReq, &mdResp)
		typeName := mdResp.TypeName

		// Fetch current schema Version.
		schemaReq := resource.SchemaRequest{}
		schemaResp := resource.SchemaResponse{}
		res.Schema(ctx, schemaReq, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("[%s] Schema returned diagnostics: %+v", typeName, schemaResp.Diagnostics)
			continue
		}
		version := schemaResp.Schema.Version

		// Only enforce for resources that have been bumped past their
		// initial declared Version. See test docstring for why Version 1
		// resources are out of scope.
		if version < 2 {
			continue
		}

		upgrader, ok := res.(resource.ResourceWithUpgradeState)
		if !ok {
			t.Errorf(
				"[%s] schema Version = %d but resource does not implement resource.ResourceWithUpgradeState. "+
					"Bumping Version without shipping an upgrader breaks every existing user's next refresh with "+
					"'Unable to Upgrade Resource State'. See v0.11.2 CHANGELOG for the failure mode this test guards.",
				typeName, version,
			)
			continue
		}

		upgraders := upgrader.UpgradeState(ctx)
		for prior := int64(1); prior < version; prior++ {
			if _, present := upgraders[prior]; !present {
				t.Errorf(
					"[%s] schema Version = %d but UpgradeState() does not return an entry for prior version %d. "+
						"Framework requires an entry for every version in [1..Version-1] — otherwise state written "+
						"by the provider release that stamped that prior version is unloadable.",
					typeName, version, prior,
				)
			}
		}
	}
}
