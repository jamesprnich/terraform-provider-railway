package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// v0.11.2 changed `pre_deploy_command` from a ListAttribute to a StringAttribute
// and correctly bumped the schema Version from 1 to 2, but shipped without an
// UpgradeState implementation. Terraform requires an upgrader for every prior
// version regardless of whether the shape difference carries real data — any
// state written by v0.11.1 or earlier has SchemaVersion: 1 stamped on it, and
// without an upgrader Terraform refuses to load it. Every user's plan/apply
// broke on the first refresh against v0.11.2. This file ships the missing
// v1→v2 upgrader.
//
// The transformation is defined for every possible v1 list shape:
//   - null list       → null string   (the only case v0.11.1 could actually
//                                       produce, because v0.11.1's Read panicked
//                                       whenever preDeployCommand was non-null)
//   - empty list      → null string   (defensive; semantically "no command")
//   - single element  → string        (the round-trip of a hand-crafted state)
//   - multi element   → strings.Join(elts, " && ")
//                                     (lossless; provider never produced this
//                                     itself, but hand-edited state or future
//                                     schema relaxation could carry >1 element,
//                                     and dropping later commands silently would
//                                     be worse than joining them)

// serviceInstanceResourceModelV1 is a byte-for-byte mirror of
// ServiceInstanceResourceModel with a single difference: pre_deploy_command is
// a List<String> rather than a String. It exists only so UpgradeState can Get
// the prior state into a struct that matches the v1 schema.
type serviceInstanceResourceModelV1 struct {
	Id                      types.String  `tfsdk:"id"`
	ServiceId               types.String  `tfsdk:"service_id"`
	EnvironmentId           types.String  `tfsdk:"environment_id"`
	SourceImage             types.String  `tfsdk:"source_image"`
	SourceRepo              types.String  `tfsdk:"source_repo"`
	RootDirectory           types.String  `tfsdk:"root_directory"`
	ConfigPath              types.String  `tfsdk:"config_path"`
	BuildCommand            types.String  `tfsdk:"build_command"`
	StartCommand            types.String  `tfsdk:"start_command"`
	Region                  types.String  `tfsdk:"region"`
	CronSchedule            types.String  `tfsdk:"cron_schedule"`
	HealthcheckPath         types.String  `tfsdk:"healthcheck_path"`
	NumReplicas             types.Int64   `tfsdk:"num_replicas"`
	VCPUs                   types.Float64 `tfsdk:"vcpus"`
	MemoryGB                types.Float64 `tfsdk:"memory_gb"`
	SleepApplication        types.Bool    `tfsdk:"sleep_application"`
	OverlapSeconds          types.Int64   `tfsdk:"overlap_seconds"`
	DrainingSeconds         types.Int64   `tfsdk:"draining_seconds"`
	HealthcheckTimeout      types.Int64   `tfsdk:"healthcheck_timeout"`
	RestartPolicyType       types.String  `tfsdk:"restart_policy_type"`
	RestartPolicyMaxRetries types.Int64   `tfsdk:"restart_policy_max_retries"`
	PreDeployCommand        types.List    `tfsdk:"pre_deploy_command"`
	WatchPatterns           types.List    `tfsdk:"watch_patterns"`
	Builder                 types.String  `tfsdk:"builder"`
	RegistryCredentials     types.Object  `tfsdk:"registry_credentials"`
}

// serviceInstanceSchemaV1 is a frozen copy of the v1 schema. Only referenced
// by UpgradeState as the PriorSchema for the v1→v2 upgrader; do not modify.
// The only difference from the current schema is Version: 1 and
// pre_deploy_command as a ListAttribute.
func serviceInstanceSchemaV1() schema.Schema {
	return schema.Schema{
		Version:     1,
		Description: "Railway service instance (v1 schema, retained for state upgrade).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"source_image": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo")),
				},
			},
			"source_repo": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(3),
				},
			},
			"root_directory": schema.StringAttribute{Optional: true},
			"config_path":    schema.StringAttribute{Optional: true},
			"build_command":  schema.StringAttribute{Optional: true},
			"start_command":  schema.StringAttribute{Optional: true},
			"region": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cron_schedule":    schema.StringAttribute{Optional: true},
			"healthcheck_path": schema.StringAttribute{Optional: true},
			"num_replicas": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"vcpus":             schema.Float64Attribute{Optional: true},
			"memory_gb":         schema.Float64Attribute{Optional: true},
			"sleep_application": schema.BoolAttribute{Optional: true},
			"overlap_seconds":   schema.Int64Attribute{Optional: true},
			"draining_seconds":  schema.Int64Attribute{Optional: true},
			"healthcheck_timeout": schema.Int64Attribute{
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"restart_policy_type": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("ALWAYS", "ON_FAILURE", "NEVER"),
				},
			},
			"restart_policy_max_retries": schema.Int64Attribute{
				Optional: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"pre_deploy_command": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"watch_patterns": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"builder": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("HEROKU", "NIXPACKS", "PAKETO", "RAILPACK"),
				},
			},
			"registry_credentials": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.UTF8LengthAtLeast(1),
						},
					},
					"password": schema.StringAttribute{
						Required:  true,
						Sensitive: true,
						Validators: []validator.String{
							stringvalidator.UTF8LengthAtLeast(1),
						},
					},
				},
			},
		},
	}
}

// upgradePreDeployCommandV1ToV2 collapses a v1 list of shell commands into a
// single v2 string. The transformation rules are documented at the top of this
// file. Extracted as a pure function so the unit test can hit every case
// without spinning up the Plugin Framework — returns Diagnostics so callers
// can bubble them into the framework's diagnostic pipeline unchanged.
func upgradePreDeployCommandV1ToV2(ctx context.Context, prior types.List) (types.String, diag.Diagnostics) {
	if prior.IsNull() || prior.IsUnknown() {
		return types.StringNull(), nil
	}

	var elements []string
	diags := prior.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return types.StringNull(), diags
	}

	if len(elements) == 0 {
		return types.StringNull(), nil
	}
	return types.StringValue(strings.Join(elements, " && ")), nil
}

// UpgradeState registers the v1→v2 upgrader. Adding a new version to the
// schema in the future means: (1) bump Version in Schema(), (2) copy the
// current schema into a serviceInstanceSchemaV(N) frozen helper, (3) register
// a new entry under key N-1 here that transforms the v(N-1) model to v(N).
//
// The reflection invariant test TestSchemaVersionsHaveUpgraders enforces that
// every prior version has a registered upgrader.
func (r *ServiceInstanceResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	priorSchemaV1 := serviceInstanceSchemaV1()

	return map[int64]resource.StateUpgrader{
		1: {
			PriorSchema: &priorSchemaV1,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorData serviceInstanceResourceModelV1
				resp.Diagnostics.Append(req.State.Get(ctx, &priorData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedPreDeploy, upgradeDiags := upgradePreDeployCommandV1ToV2(ctx, priorData.PreDeployCommand)
				resp.Diagnostics.Append(upgradeDiags...)
				if resp.Diagnostics.HasError() {
					return
				}

				newData := ServiceInstanceResourceModel{
					Id:                      priorData.Id,
					ServiceId:               priorData.ServiceId,
					EnvironmentId:           priorData.EnvironmentId,
					SourceImage:             priorData.SourceImage,
					SourceRepo:              priorData.SourceRepo,
					RootDirectory:           priorData.RootDirectory,
					ConfigPath:              priorData.ConfigPath,
					BuildCommand:            priorData.BuildCommand,
					StartCommand:            priorData.StartCommand,
					Region:                  priorData.Region,
					CronSchedule:            priorData.CronSchedule,
					HealthcheckPath:         priorData.HealthcheckPath,
					NumReplicas:             priorData.NumReplicas,
					VCPUs:                   priorData.VCPUs,
					MemoryGB:                priorData.MemoryGB,
					SleepApplication:        priorData.SleepApplication,
					OverlapSeconds:          priorData.OverlapSeconds,
					DrainingSeconds:         priorData.DrainingSeconds,
					HealthcheckTimeout:      priorData.HealthcheckTimeout,
					RestartPolicyType:       priorData.RestartPolicyType,
					RestartPolicyMaxRetries: priorData.RestartPolicyMaxRetries,
					PreDeployCommand:        upgradedPreDeploy,
					WatchPatterns:           priorData.WatchPatterns,
					Builder:                 priorData.Builder,
					RegistryCredentials:     priorData.RegistryCredentials,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, newData)...)
			},
		},
	}
}
