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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &PrivateNetworkResource{}
var _ resource.ResourceWithImportState = &PrivateNetworkResource{}

func NewPrivateNetworkResource() resource.Resource {
	return &PrivateNetworkResource{}
}

type PrivateNetworkResource struct {
	client *graphql.Client
}

type PrivateNetworkResourceModel struct {
	Id            types.String `tfsdk:"id"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	Name          types.String `tfsdk:"name"`
	DnsName       types.String `tfsdk:"dns_name"`
	NetworkId     types.Int64  `tfsdk:"network_id"`
	Tags          types.List   `tfsdk:"tags"`
}

func (r *PrivateNetworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network"
}

func (r *PrivateNetworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway private network. Creates a private network in a specific environment for internal service-to-service communication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the private network (publicId).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the private network belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the private network belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the private network.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "DNS name of the private network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric network identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.ListAttribute{
				MarkdownDescription: "Tags associated with the private network.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *PrivateNetworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrivateNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *PrivateNetworkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		tags = []string{}
	}

	input := PrivateNetworkCreateOrGetInput{
		EnvironmentId: data.EnvironmentId.ValueString(),
		Name:          data.Name.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
		Tags:          tags,
	}

	response, err := createOrGetPrivateNetwork(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create private network, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a private network")

	network := response.PrivateNetworkCreateOrGet

	data.Id = types.StringValue(network.PrivateNetworkFields.PublicId)
	data.DnsName = types.StringValue(network.PrivateNetworkFields.DnsName)
	data.NetworkId = types.Int64Value(network.PrivateNetworkFields.NetworkId)
	data.Name = types.StringValue(network.PrivateNetworkFields.Name)

	tagList, diags := types.ListValueFrom(ctx, types.StringType, network.PrivateNetworkFields.Tags)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Tags = tagList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *PrivateNetworkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readPrivateNetworkState(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The API is create-or-get; name changes require destroy/recreate (RequiresReplace).
	// Tags-only changes re-call createOrGet which is idempotent.
	var data *PrivateNetworkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		tags = []string{}
	}

	input := PrivateNetworkCreateOrGetInput{
		EnvironmentId: data.EnvironmentId.ValueString(),
		Name:          data.Name.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
		Tags:          tags,
	}

	response, err := createOrGetPrivateNetwork(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update private network, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated private network")

	network := response.PrivateNetworkCreateOrGet

	data.Id = types.StringValue(network.PrivateNetworkFields.PublicId)
	data.DnsName = types.StringValue(network.PrivateNetworkFields.DnsName)
	data.NetworkId = types.Int64Value(network.PrivateNetworkFields.NetworkId)
	data.Name = types.StringValue(network.PrivateNetworkFields.Name)

	tagList, diags := types.ListValueFrom(ctx, types.StringType, network.PrivateNetworkFields.Tags)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Tags = tagList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *PrivateNetworkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deletePrivateNetworksForEnvironment(ctx, *r.client, data.EnvironmentId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete private networks, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted private networks for environment")
}

func (r *PrivateNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: environment_id:network_public_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// readPrivateNetworkState queries the environment's private networks and populates the model.
func (r *PrivateNetworkResource) readPrivateNetworkState(ctx context.Context, data *PrivateNetworkResourceModel) error {
	response, err := getPrivateNetworks(ctx, *r.client, data.EnvironmentId.ValueString())

	if err != nil {
		return err
	}

	for _, network := range response.PrivateNetworks {
		if network.PrivateNetworkFields.PublicId != data.Id.ValueString() {
			continue
		}

		data.Name = types.StringValue(network.PrivateNetworkFields.Name)
		data.DnsName = types.StringValue(network.PrivateNetworkFields.DnsName)
		data.NetworkId = types.Int64Value(network.PrivateNetworkFields.NetworkId)
		data.ProjectId = types.StringValue(network.PrivateNetworkFields.ProjectId)

		tagList, diags := types.ListValueFrom(ctx, types.StringType, network.PrivateNetworkFields.Tags)
		if diags.HasError() {
			return fmt.Errorf("unable to convert tags: %s", diags.Errors())
		}

		data.Tags = tagList
		return nil
	}

	return fmt.Errorf("private network %s not found in environment %s", data.Id.ValueString(), data.EnvironmentId.ValueString())
}
