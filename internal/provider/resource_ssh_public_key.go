package provider

import (
	"context"
	"fmt"

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

var _ resource.Resource = &SshPublicKeyResource{}
var _ resource.ResourceWithImportState = &SshPublicKeyResource{}

func NewSshPublicKeyResource() resource.Resource {
	return &SshPublicKeyResource{}
}

type SshPublicKeyResource struct {
	client *graphql.Client
}

type SshPublicKeyResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
}

func (r *SshPublicKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_public_key"
}

func (r *SshPublicKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway SSH public key. Registers an SSH key against a workspace (or the authenticated user if `workspace_id` is omitted) for use with Railway features that require SSH authentication.",
		Description:         "Railway SSH public key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the SSH key.",
				Description:         "Identifier of the SSH key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Friendly name for the SSH key.",
				Description:         "Friendly name for the SSH key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "OpenSSH-format public key (e.g. `ssh-ed25519 AAAA... user@host`).",
				Description:         "OpenSSH-format public key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "Server-computed fingerprint of the public key.",
				Description:         "Server-computed fingerprint of the public key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace the key belongs to. If omitted, the key is registered against the authenticated user.",
				Description:         "Identifier of the workspace the key belongs to.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *SshPublicKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SshPublicKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *SshPublicKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := SshPublicKeyCreateInput{
		Name:      data.Name.ValueString(),
		PublicKey: data.PublicKey.ValueString(),
	}
	if !data.WorkspaceId.IsNull() && !data.WorkspaceId.IsUnknown() {
		v := data.WorkspaceId.ValueString()
		input.WorkspaceId = &v
	}

	response, err := createSshPublicKey(ctx, *r.client, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create SSH public key (name=%s), got error: %s", data.Name.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created an SSH public key")

	key := response.SshPublicKeyCreate
	data.Id = types.StringValue(key.Id)
	data.Name = types.StringValue(key.Name)
	data.PublicKey = types.StringValue(key.PublicKey)
	data.Fingerprint = types.StringValue(key.Fingerprint)
	if key.WorkspaceId != "" {
		data.WorkspaceId = types.StringValue(key.WorkspaceId)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SshPublicKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *SshPublicKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceId := data.WorkspaceId.ValueString() // empty string is fine — query treats it as "personal"
	response, err := getSshPublicKeys(ctx, *r.client, workspaceId)

	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SSH public keys, got error: %s", err))
		return
	}

	var found bool
	for _, edge := range response.SshPublicKeys.Edges {
		if edge.Node.Id == data.Id.ValueString() {
			data.Name = types.StringValue(edge.Node.Name)
			data.PublicKey = types.StringValue(edge.Node.PublicKey)
			data.Fingerprint = types.StringValue(edge.Node.Fingerprint)
			if edge.Node.WorkspaceId != "" {
				data.WorkspaceId = types.StringValue(edge.Node.WorkspaceId)
			}
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

func (r *SshPublicKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "SSH public keys are immutable; use replace instead.")
}

func (r *SshPublicKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *SshPublicKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteSshPublicKey(ctx, *r.client, data.Id.ValueString())
	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete SSH public key (id=%s), got error: %s", data.Id.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "deleted an SSH public key")
}

func (r *SshPublicKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
