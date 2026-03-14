package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DeploymentTriggerResource{}
var _ resource.ResourceWithImportState = &DeploymentTriggerResource{}

func NewDeploymentTriggerResource() resource.Resource {
	return &DeploymentTriggerResource{}
}

type DeploymentTriggerResource struct {
	client *graphql.Client
}

type DeploymentTriggerResourceModel struct {
	Id            types.String `tfsdk:"id"`
	ServiceId     types.String `tfsdk:"service_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Repository    types.String `tfsdk:"repository"`
	Branch        types.String `tfsdk:"branch"`
	CheckSuites   types.Bool   `tfsdk:"check_suites"`
	Provider      types.String `tfsdk:"source_provider"`
	RootDirectory types.String `tfsdk:"root_directory"`
}

func (r *DeploymentTriggerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_trigger"
}

func (r *DeploymentTriggerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway deployment trigger. Connects a source code repository to a service so that pushes to a branch automatically trigger deployments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the deployment trigger.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the deployment trigger belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the deployment trigger belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the deployment trigger belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: "Repository to watch for changes (e.g. \"owner/repo\").",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "Branch to watch for changes.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"check_suites": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for check suites to pass before deploying.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"source_provider": schema.StringAttribute{
				MarkdownDescription: "Source provider (e.g. \"github\").",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Root directory within the repository. Only relevant for monorepos.",
				Optional:            true,
			},
		},
	}
}

func (r *DeploymentTriggerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *DeploymentTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := DeploymentTriggerCreateInput{
		Branch:        data.Branch.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
		Provider:      data.Provider.ValueString(),
		Repository:    data.Repository.ValueString(),
		ServiceId:     data.ServiceId.ValueString(),
	}

	if !data.CheckSuites.IsNull() && !data.CheckSuites.IsUnknown() {
		checkSuites := data.CheckSuites.ValueBool()
		input.CheckSuites = &checkSuites
	}

	if !data.RootDirectory.IsNull() && !data.RootDirectory.IsUnknown() {
		rootDir := data.RootDirectory.ValueString()
		input.RootDirectory = &rootDir
	}

	response, err := createDeploymentTrigger(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create deployment trigger (service_id=%s, environment_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), err))
		return
	}

	tflog.Trace(ctx, "created a deployment trigger")

	trigger := response.DeploymentTriggerCreate.DeploymentTrigger

	data.Id = types.StringValue(trigger.Id)
	data.Branch = types.StringValue(trigger.Branch)
	data.CheckSuites = types.BoolValue(trigger.CheckSuites)
	data.EnvironmentId = types.StringValue(trigger.EnvironmentId)
	data.ProjectId = types.StringValue(trigger.ProjectId)
	data.Provider = types.StringValue(trigger.Provider)
	data.Repository = types.StringValue(trigger.Repository)

	if trigger.ServiceId != "" {
		data.ServiceId = types.StringValue(trigger.ServiceId)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readTriggerState(ctx, data)

	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment trigger (service_id=%s, environment_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *DeploymentTriggerResourceModel
	var state *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := DeploymentTriggerUpdateInput{}
	needsUpdate := false

	if data.Branch.ValueString() != state.Branch.ValueString() {
		branch := data.Branch.ValueString()
		input.Branch = &branch
		needsUpdate = true
	}

	if !data.CheckSuites.Equal(state.CheckSuites) {
		checkSuites := data.CheckSuites.ValueBool()
		input.CheckSuites = &checkSuites
		needsUpdate = true
	}

	if data.Repository.ValueString() != state.Repository.ValueString() {
		repo := data.Repository.ValueString()
		input.Repository = &repo
		needsUpdate = true
	}

	if !data.RootDirectory.Equal(state.RootDirectory) {
		if !data.RootDirectory.IsNull() {
			rootDir := data.RootDirectory.ValueString()
			input.RootDirectory = &rootDir
		} else {
			empty := ""
			input.RootDirectory = &empty
		}
		needsUpdate = true
	}

	if needsUpdate {
		response, err := updateDeploymentTrigger(ctx, *r.client, state.Id.ValueString(), input)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update deployment trigger, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "updated a deployment trigger")

		trigger := response.DeploymentTriggerUpdate.DeploymentTrigger

		data.Id = types.StringValue(trigger.Id)
		data.Branch = types.StringValue(trigger.Branch)
		data.CheckSuites = types.BoolValue(trigger.CheckSuites)
		data.EnvironmentId = types.StringValue(trigger.EnvironmentId)
		data.ProjectId = types.StringValue(trigger.ProjectId)
		data.Provider = types.StringValue(trigger.Provider)
		data.Repository = types.StringValue(trigger.Repository)

		if trigger.ServiceId != "" {
			data.ServiceId = types.StringValue(trigger.ServiceId)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteDeploymentTrigger(ctx, *r.client, data.Id.ValueString())

	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete deployment trigger, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a deployment trigger")
}

func (r *DeploymentTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:environment_id:service_id:trigger_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// readTriggerState queries the deployment triggers for a service/environment and populates the model.
func (r *DeploymentTriggerResource) readTriggerState(ctx context.Context, data *DeploymentTriggerResourceModel) error {
	response, err := getDeploymentTriggers(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.ProjectId.ValueString(),
		data.ServiceId.ValueString(),
	)

	if err != nil {
		return err
	}

	for _, edge := range response.DeploymentTriggers.Edges {
		trigger := edge.Node.DeploymentTrigger

		if trigger.Id != data.Id.ValueString() {
			continue
		}

		data.Branch = types.StringValue(trigger.Branch)
		data.CheckSuites = types.BoolValue(trigger.CheckSuites)
		data.EnvironmentId = types.StringValue(trigger.EnvironmentId)
		data.ProjectId = types.StringValue(trigger.ProjectId)
		data.Provider = types.StringValue(trigger.Provider)
		data.Repository = types.StringValue(trigger.Repository)

		if trigger.ServiceId != "" {
			data.ServiceId = types.StringValue(trigger.ServiceId)
		}

		return nil
	}

	return &NotFoundError{ResourceType: "deployment trigger", Id: data.Id.ValueString()}
}
