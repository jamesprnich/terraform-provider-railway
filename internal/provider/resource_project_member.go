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

var _ resource.Resource = &ProjectMemberResource{}
var _ resource.ResourceWithImportState = &ProjectMemberResource{}

func NewProjectMemberResource() resource.Resource {
	return &ProjectMemberResource{}
}

type ProjectMemberResource struct {
	client *graphql.Client
}

type ProjectMemberResourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	UserId    types.String `tfsdk:"user_id"`
	Role      types.String `tfsdk:"role"`
	Email     types.String `tfsdk:"email"`
	Name      types.String `tfsdk:"name"`
}

func (r *ProjectMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_member"
}

func (r *ProjectMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway project membership. Adds a user to a project with a given role.",
		Description:         "Railway project membership.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project membership.",
				Description:         "Identifier of the project membership.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project.",
				Description:         "Identifier of the project.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the user to add as a member.",
				Description:         "Identifier of the user to add as a member.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role assigned to the member. One of `ADMIN`, `MEMBER`, `VIEWER`.",
				Description:         "Role assigned to the member. One of ADMIN, MEMBER, VIEWER.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ADMIN", "MEMBER", "VIEWER"),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address of the member.",
				Description:         "Email address of the member.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the member.",
				Description:         "Display name of the member.",
				Computed:            true,
			},
		},
	}
}

func (r *ProjectMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := providerDataFrom(req.ProviderData, &resp.Diagnostics)
	if data == nil {
		return
	}

	r.client = data.Client
}

func (r *ProjectMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ProjectMemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := ProjectMemberAddInput{
		ProjectId: data.ProjectId.ValueString(),
		UserId:    data.UserId.ValueString(),
		Role:      ProjectRole(data.Role.ValueString()),
	}

	response, err := addProjectMember(ctx, *r.client, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add project member (project_id=%s, user_id=%s), got error: %s", data.ProjectId.ValueString(), data.UserId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "added a project member")

	member := response.ProjectMemberAdd
	data.Id = types.StringValue(member.Id)
	data.Role = types.StringValue(string(member.Role))
	data.Email = types.StringValue(member.Email)
	data.Name = types.StringValue(member.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ProjectMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getProjectMembers(ctx, *r.client, data.ProjectId.ValueString())

	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project members (project_id=%s), got error: %s", data.ProjectId.ValueString(), err))
		return
	}

	var found bool
	for _, member := range response.ProjectMembers {
		if member.Id == data.Id.ValueString() || member.Id == data.UserId.ValueString() {
			data.Id = types.StringValue(member.Id)
			data.Role = types.StringValue(string(member.Role))
			data.Email = types.StringValue(member.Email)
			data.Name = types.StringValue(member.Name)
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

func (r *ProjectMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ProjectMemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := ProjectMemberUpdateInput{
		ProjectId: data.ProjectId.ValueString(),
		UserId:    data.UserId.ValueString(),
		Role:      ProjectRole(data.Role.ValueString()),
	}

	response, err := updateProjectMember(ctx, *r.client, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project member (project_id=%s, user_id=%s), got error: %s", data.ProjectId.ValueString(), data.UserId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "updated a project member")

	member := response.ProjectMemberUpdate
	data.Role = types.StringValue(string(member.Role))
	data.Email = types.StringValue(member.Email)
	data.Name = types.StringValue(member.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ProjectMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := ProjectMemberRemoveInput{
		ProjectId: data.ProjectId.ValueString(),
		UserId:    data.UserId.ValueString(),
	}

	_, err := removeProjectMember(ctx, *r.client, input)
	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove project member (project_id=%s, user_id=%s), got error: %s", data.ProjectId.ValueString(), data.UserId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "removed a project member")
}

func (r *ProjectMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:user_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
