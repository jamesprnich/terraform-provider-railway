package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}
var _ resource.ResourceWithModifyPlan = &EnvironmentResource{}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

type EnvironmentResource struct {
	client           *graphql.Client
	strictEnvScoping bool
}

type EnvironmentResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	ProjectId           types.String `tfsdk:"project_id"`
	SourceEnvironmentId types.String `tfsdk:"source_environment_id"`
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 2,
		MarkdownDescription: "Additional Railway environment created as a fork of another environment (typically the project's " +
			"empty default environment). Under `strict_env_scoping = true` (provider default), `source_environment_id` " +
			"is required — a non-fork environment breaks per-environment resource scoping and is rejected at plan time.",
		Description: "Additional Railway environment created as a fork of another environment (typically the project's " +
			"empty default environment).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment.",
				Description:         "Identifier of the environment.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment.",
				Description:         "Name of the environment.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the environment belongs to.",
				Description:         "Identifier of the project the environment belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"source_environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment this environment is forked from. Required under " +
					"strict env-scoping (provider default). Forking is the mechanism Railway uses to make " +
					"`serviceCreate` and `serviceDelete` scope to a single environment — a non-fork environment " +
					"causes services to be created across every non-fork environment in the project. Setting this " +
					"to the project's empty default environment (`railway_project.default_environment.id`) is the " +
					"safe pattern. Cannot be changed after creation.\n\n" +
					"~> **Never fork a real environment.** Railway's fork semantic copies every service, volume, " +
					"variable, and configuration from the source environment into the new one. Always fork the " +
					"project's empty default environment (`core` in the recommended layout). Forking `dev` to " +
					"create `prd` will silently duplicate `dev`'s state into `prd`.",
				Description: "Identifier of the environment this environment is forked from. Required under strict env-scoping.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
		},
	}
}

func (r *EnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := providerDataFrom(req.ProviderData, &resp.Diagnostics)
	if data == nil {
		return
	}

	r.client = data.Client
	r.strictEnvScoping = data.StrictEnvScoping
}

// ModifyPlan runs the strict-env-scoping check at plan time. See
// ServiceResource.ModifyPlan for the reasoning behind reading config
// (not plan) and the null-vs-unknown distinction.
func (r *EnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config EnvironmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.strictEnvScoping && config.SourceEnvironmentId.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("source_environment_id"),
			"source_environment_id is required under strict env-scoping",
			"The Railway provider defaults to strict env-scoping (`strict_env_scoping = true`) to enforce a "+
				"fork-based multi-environment layout. Set `source_environment_id` to the id of an existing "+
				"environment (typically `railway_project.default_environment.id`) so this environment becomes a "+
				"fork of it. To opt out and create a non-fork environment (Railway's default), set "+
				"`strict_env_scoping = false` on the provider block — you own the leak surface.",
		)
	}
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *EnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Under strict mode, source_environment_id must be set in config. Since
	// the attribute is Optional+Computed, an unset config surfaces here as
	// Unknown (not Null) — treat both as "the user didn't supply one."
	if r.strictEnvScoping && (data.SourceEnvironmentId.IsNull() || data.SourceEnvironmentId.IsUnknown()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("source_environment_id"),
			"source_environment_id is required under strict env-scoping",
			"The Railway provider defaults to strict env-scoping (`strict_env_scoping = true`) to enforce a "+
				"fork-based multi-environment layout. Set `source_environment_id` to the id of an existing "+
				"environment (typically `railway_project.default_environment.id`) so this environment becomes a "+
				"fork of it. To opt out and create a non-fork environment (Railway's default), set "+
				"`strict_env_scoping = false` on the provider block — you own the leak surface.",
		)
		return
	}

	// Explicit values for every non-pointer bool on EnvironmentCreateInput.
	// StageInitialChanges: false means changes commit immediately rather than
	// sitting as unmerged changes the user has to click "apply" on. The other
	// three defaults ensure this environment behaves like a normal, persistent
	// environment. Callers cannot currently override these — that would be a
	// separate schema decision.
	input := EnvironmentCreateInput{
		Name:                     data.Name.ValueString(),
		ProjectId:                data.ProjectId.ValueString(),
		StageInitialChanges:      false,
		Ephemeral:                false,
		SkipInitialDeploys:       false,
		ApplyChangesInBackground: false,
	}

	// Only pass sourceEnvironmentId when the user actually supplied one.
	// An Optional+Computed attribute with no config value surfaces here as
	// Unknown, not Null; treating Unknown as "set to empty" would send
	// sourceEnvironmentId="" to Railway, which errors with "Environment not
	// found". Under permissive mode this is the path that creates a non-fork
	// environment — the field must be omitted, not empty.
	if !data.SourceEnvironmentId.IsNull() && !data.SourceEnvironmentId.IsUnknown() {
		src := data.SourceEnvironmentId.ValueString()
		input.SourceEnvironmentId = &src
	}

	// Railway enforces a per-user environment-creation cooldown ("only one
	// environment can be created per user every 30s"). retryOnCooldownContext
	// waits it out transparently.
	var response *createEnvironmentResponse
	err := retryOnCooldownContext(ctx, 90*time.Second, func() error {
		var callErr error
		response, callErr = createEnvironment(ctx, *r.client, input)
		return callErr
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment %q (project_id=%s), got error: %s", data.Name.ValueString(), data.ProjectId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created an environment")

	environment := response.EnvironmentCreate.Environment

	data.Id = types.StringValue(environment.Id)
	data.Name = types.StringValue(environment.Name)
	data.ProjectId = types.StringValue(environment.ProjectId)
	data.SourceEnvironmentId = readSourceEnvironmentId(environment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *EnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Use getEnvironments (project list) instead of getEnvironment (by ID).
	// The individual environment(id:) query can return stale data for deleted
	// environments, while the project list is authoritative.
	envsResponse, err := getEnvironments(ctx, *r.client, data.ProjectId.ValueString())

	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environments for project (id=%s, environment=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.Name.ValueString(), data.ProjectId.ValueString(), err))
		return
	}

	var found bool
	for _, edge := range envsResponse.Environments.Edges {
		env := edge.Node.Environment
		if env.Id == data.Id.ValueString() {
			data.Id = types.StringValue(env.Id)
			data.Name = types.StringValue(env.Name)
			data.ProjectId = types.StringValue(env.ProjectId)
			data.SourceEnvironmentId = readSourceEnvironmentId(env)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *EnvironmentResourceModel
	var state *EnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		input := EnvironmentRenameInput{
			Name: plan.Name.ValueString(),
		}

		response, err := renameEnvironment(ctx, *r.client, state.Id.ValueString(), input)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to rename environment, got error: %s", err))
			return
		}

		tflog.Debug(ctx, "updated an environment")

		plan.Id = types.StringValue(response.EnvironmentRename.Id)
		plan.Name = types.StringValue(response.EnvironmentRename.Name)
		plan.ProjectId = types.StringValue(response.EnvironmentRename.ProjectId)
		plan.SourceEnvironmentId = readSourceEnvironmentId(response.EnvironmentRename.Environment)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *EnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Verify the environment still exists before attempting deletion.
	// getEnvironment(id) can return stale data, so check the project list.
	envsResponse, err := getEnvironments(ctx, *r.client, data.ProjectId.ValueString())
	if err == nil {
		found := false
		for _, edge := range envsResponse.Environments.Edges {
			if edge.Node.Id == data.Id.ValueString() {
				found = true
				break
			}
		}
		if !found {
			// Already deleted externally
			return
		}
	}

	_, err = deleteEnvironment(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		if isNotFoundOrGone(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "deleted an environment")
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:name. Got: %q", req.ID),
		)

		return
	}

	environmentId, err := findEnvironment(ctx, *r.client, parts[0], parts[1])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
}

func findEnvironment(ctx context.Context, client graphql.Client, projectId string, name string) (*string, error) {
	response, err := getEnvironments(ctx, client, projectId)

	if err != nil {
		return nil, err
	}

	for _, environment := range response.Environments.Edges {
		if environment.Node.Name == name {
			return &environment.Node.Id, nil
		}
	}

	return nil, fmt.Errorf("environment doesn't exist in the project")
}

// readSourceEnvironmentId converts the sourceEnvironment.id field on an
// Environment fragment into a types.String, treating an empty id as "no
// source" (non-fork environment).
func readSourceEnvironmentId(env Environment) types.String {
	if env.SourceEnvironment.Id == "" {
		return types.StringNull()
	}
	return types.StringValue(env.SourceEnvironment.Id)
}
