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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ProjectTokenResource{}
var _ resource.ResourceWithImportState = &ProjectTokenResource{}

func NewProjectTokenResource() resource.Resource {
	return &ProjectTokenResource{}
}

type ProjectTokenResource struct {
	client *graphql.Client
}

type ProjectTokenResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	Token         types.String `tfsdk:"token"`
}

func (r *ProjectTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_token"
}

func (r *ProjectTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway project token. Generates a scoped deploy token for CI/CD pipelines.\n\n~> **Note:** The `token` attribute is only available at creation time. Railway does not return the raw token on subsequent reads. Store it in your CI secret manager immediately after `tofu apply`.",
		Description:         "Railway project token for CI/CD authentication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project token.",
				Description:         "Identifier of the project token.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project token.",
				Description:         "Name of the project token.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the token belongs to.",
				Description:         "Identifier of the project the token belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the token has access to.",
				Description:         "Identifier of the environment the token has access to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The raw token value. Only populated at creation; null on import or read.",
				Description:         "The raw token value. Only populated at creation; null on import or read.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProjectTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ProjectTokenResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ProjectTokenCreateInput{
		Name:          data.Name.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
	}

	response, err := createProjectToken(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project token (project_id=%s), got error: %s", data.ProjectId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created a project token")

	// projectTokenCreate returns the raw token string only.
	// We must query projectTokens to find the ID by name match.
	data.Token = types.StringValue(response.ProjectTokenCreate)

	// Save token to state immediately — it can't be retrieved later.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tokenId, err := r.findTokenIdByName(ctx, data.ProjectId.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Project token was created but could not be located by name (project_id=%s, name=%s): %s", data.ProjectId.ValueString(), data.Name.ValueString(), err))
		return
	}

	data.Id = types.StringValue(tokenId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ProjectTokenResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getProjectTokens(ctx, *r.client, data.ProjectId.ValueString())

	if isNotFound(err) {
		tflog.Info(ctx, "project tokens not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project tokens (project_id=%s), got error: %s", data.ProjectId.ValueString(), err))
		return
	}

	var found bool
	for _, edge := range response.ProjectTokens.Edges {
		if edge.Node.Id == data.Id.ValueString() {
			data.Name = types.StringValue(edge.Node.Name)
			data.ProjectId = types.StringValue(edge.Node.ProjectId)
			data.EnvironmentId = types.StringValue(edge.Node.EnvironmentId)
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

func (r *ProjectTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes use RequiresReplace, so Update is unreachable.
	resp.Diagnostics.AddError("Update not supported", "Project tokens are immutable; use replace instead.")
}

func (r *ProjectTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ProjectTokenResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteProjectToken(ctx, *r.client, data.Id.ValueString())

	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project token (id=%s), got error: %s", data.Id.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "deleted a project token")
}

func (r *ProjectTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:token_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *ProjectTokenResource) findTokenIdByName(ctx context.Context, projectId, name string) (string, error) {
	response, err := getProjectTokens(ctx, *r.client, projectId)
	if err != nil {
		return "", err
	}

	for _, edge := range response.ProjectTokens.Edges {
		if edge.Node.Name == name {
			return edge.Node.Id, nil
		}
	}

	return "", fmt.Errorf("token with name %q not found in project %s", name, projectId)
}
