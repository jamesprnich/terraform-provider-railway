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

var _ resource.Resource = &TrustedDomainResource{}
var _ resource.ResourceWithImportState = &TrustedDomainResource{}

func NewTrustedDomainResource() resource.Resource {
	return &TrustedDomainResource{}
}

type TrustedDomainResource struct {
	client *graphql.Client
}

type TrustedDomainResourceModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	DomainName  types.String `tfsdk:"domain_name"`
	Role        types.String `tfsdk:"role"`
	Status      types.String `tfsdk:"status"`
}

func (r *TrustedDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trusted_domain"
}

func (r *TrustedDomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway workspace trusted domain. Used for SSO and access control. The domain must be verified via DNS before it is honoured.",
		Description:         "Railway workspace trusted domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the trusted domain.",
				Description:         "Identifier of the trusted domain.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace the trusted domain belongs to.",
				Description:         "Identifier of the workspace the trusted domain belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "The fully-qualified domain name to trust (e.g. `example.com`).",
				Description:         "The fully-qualified domain name to trust.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role assigned to users authenticating from this domain (e.g. `MEMBER`).",
				Description:         "Role assigned to users authenticating from this domain.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Verification status of the trusted domain. One of `PENDING`, `VERIFIED`, `FAILED`.",
				Description:         "Verification status of the trusted domain.",
				Computed:            true,
			},
		},
	}
}

func (r *TrustedDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := providerDataFrom(req.ProviderData, &resp.Diagnostics)
	if data == nil {
		return
	}

	r.client = data.Client
}

func (r *TrustedDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *TrustedDomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := WorkspaceTrustedDomainCreateInput{
		WorkspaceId: data.WorkspaceId.ValueString(),
		DomainName:  data.DomainName.ValueString(),
		Role:        data.Role.ValueString(),
	}

	response, err := createTrustedDomain(ctx, *r.client, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create trusted domain (workspace_id=%s, domain=%s), got error: %s", data.WorkspaceId.ValueString(), data.DomainName.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created a trusted domain")

	td := response.TrustedDomainCreate
	data.Id = types.StringValue(td.Id)
	data.WorkspaceId = types.StringValue(td.WorkspaceId)
	data.DomainName = types.StringValue(td.DomainName)
	data.Role = types.StringValue(td.Role)
	data.Status = types.StringValue(string(td.Status))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TrustedDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *TrustedDomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getTrustedDomains(ctx, *r.client, data.WorkspaceId.ValueString())

	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read trusted domains (workspace_id=%s), got error: %s", data.WorkspaceId.ValueString(), err))
		return
	}

	var found bool
	for _, edge := range response.TrustedDomains.Edges {
		if edge.Node.Id == data.Id.ValueString() {
			data.WorkspaceId = types.StringValue(edge.Node.WorkspaceId)
			data.DomainName = types.StringValue(edge.Node.DomainName)
			data.Role = types.StringValue(edge.Node.Role)
			data.Status = types.StringValue(string(edge.Node.Status))
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

func (r *TrustedDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes use RequiresReplace; Update should be unreachable.
	resp.Diagnostics.AddError("Update not supported", "Trusted domains are immutable; use replace instead.")
}

func (r *TrustedDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *TrustedDomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteTrustedDomain(ctx, *r.client, data.Id.ValueString())
	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete trusted domain (id=%s), got error: %s", data.Id.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "deleted a trusted domain")
}

func (r *TrustedDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_id:trusted_domain_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
