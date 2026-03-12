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

var _ resource.Resource = &PrivateNetworkEndpointResource{}
var _ resource.ResourceWithImportState = &PrivateNetworkEndpointResource{}

func NewPrivateNetworkEndpointResource() resource.Resource {
	return &PrivateNetworkEndpointResource{}
}

type PrivateNetworkEndpointResource struct {
	client *graphql.Client
}

type PrivateNetworkEndpointResourceModel struct {
	Id               types.String `tfsdk:"id"`
	PrivateNetworkId types.String `tfsdk:"private_network_id"`
	ServiceId        types.String `tfsdk:"service_id"`
	EnvironmentId    types.String `tfsdk:"environment_id"`
	ServiceName      types.String `tfsdk:"service_name"`
	DnsName          types.String `tfsdk:"dns_name"`
	PrivateIps       types.List   `tfsdk:"private_ips"`
	Tags             types.List   `tfsdk:"tags"`
}

func (r *PrivateNetworkEndpointResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network_endpoint"
}

func (r *PrivateNetworkEndpointResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway private network endpoint. Connects a service to a private network, enabling internal service-to-service communication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the private network endpoint (publicId).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_network_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the private network to connect to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service to connect.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_name": schema.StringAttribute{
				MarkdownDescription: "Name of the service (required for the create input).",
				Required:            true,
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "DNS name of the endpoint within the private network. Can be changed via rename.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_ips": schema.ListAttribute{
				MarkdownDescription: "List of private IP addresses assigned to the endpoint.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"tags": schema.ListAttribute{
				MarkdownDescription: "Tags associated with the private network endpoint.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *PrivateNetworkEndpointResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrivateNetworkEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *PrivateNetworkEndpointResourceModel

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

	// Capture the planned dns_name before overwriting with API response
	planDnsName := data.DnsName

	input := PrivateNetworkEndpointCreateOrGetInput{
		EnvironmentId:    data.EnvironmentId.ValueString(),
		PrivateNetworkId: data.PrivateNetworkId.ValueString(),
		ServiceId:        data.ServiceId.ValueString(),
		ServiceName:      data.ServiceName.ValueString(),
		Tags:             tags,
	}

	response, err := createOrGetPrivateNetworkEndpoint(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create private network endpoint, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a private network endpoint")

	endpoint := response.PrivateNetworkEndpointCreateOrGet

	data.Id = types.StringValue(endpoint.PrivateNetworkEndpointFields.PublicId)
	data.DnsName = types.StringValue(endpoint.PrivateNetworkEndpointFields.DnsName)

	ipList, diags := types.ListValueFrom(ctx, types.StringType, endpoint.PrivateNetworkEndpointFields.PrivateIps)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.PrivateIps = ipList

	tagList, diags := types.ListValueFrom(ctx, types.StringType, endpoint.PrivateNetworkEndpointFields.Tags)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Tags = tagList

	// Save state immediately so Terraform tracks this resource.
	// If the rename step fails, the resource will be tainted
	// and scheduled for destroy+recreate on the next apply.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If dns_name was specified in config and differs from the default, rename it
	if !planDnsName.IsNull() && !planDnsName.IsUnknown() {
		if planDnsName.ValueString() != endpoint.PrivateNetworkEndpointFields.DnsName {
			_, err := renamePrivateNetworkEndpoint(ctx, *r.client, planDnsName.ValueString(), data.Id.ValueString(), data.PrivateNetworkId.ValueString())

			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to rename private network endpoint, got error: %s", err))
				return
			}

			data.DnsName = planDnsName

			tflog.Trace(ctx, "renamed private network endpoint")
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *PrivateNetworkEndpointResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getPrivateNetworkEndpoint(ctx, *r.client, data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network endpoint, got error: %s", err))
		return
	}

	endpoint := response.PrivateNetworkEndpoint

	if endpoint.PrivateNetworkEndpointFields.PublicId == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(endpoint.PrivateNetworkEndpointFields.PublicId)
	data.DnsName = types.StringValue(endpoint.PrivateNetworkEndpointFields.DnsName)

	ipList, diags := types.ListValueFrom(ctx, types.StringType, endpoint.PrivateNetworkEndpointFields.PrivateIps)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.PrivateIps = ipList

	tagList, diags := types.ListValueFrom(ctx, types.StringType, endpoint.PrivateNetworkEndpointFields.Tags)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Tags = tagList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *PrivateNetworkEndpointResourceModel
	var state *PrivateNetworkEndpointResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = state.Id

	// Rename the endpoint if dns_name changed
	if !data.DnsName.Equal(state.DnsName) && !data.DnsName.IsNull() && !data.DnsName.IsUnknown() {
		_, err := renamePrivateNetworkEndpoint(ctx, *r.client, data.DnsName.ValueString(), data.Id.ValueString(), data.PrivateNetworkId.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to rename private network endpoint, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "renamed private network endpoint")
	}

	// Read back the state
	response, err := getPrivateNetworkEndpoint(ctx, *r.client, data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network endpoint after update, got error: %s", err))
		return
	}

	endpoint := response.PrivateNetworkEndpoint

	if endpoint.PrivateNetworkEndpointFields.PublicId == "" {
		resp.Diagnostics.AddError("Client Error", "Private network endpoint not found after update")
		return
	}

	data.DnsName = types.StringValue(endpoint.PrivateNetworkEndpointFields.DnsName)

	ipList, diags := types.ListValueFrom(ctx, types.StringType, endpoint.PrivateNetworkEndpointFields.PrivateIps)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.PrivateIps = ipList

	tagList, diags := types.ListValueFrom(ctx, types.StringType, endpoint.PrivateNetworkEndpointFields.Tags)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Tags = tagList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *PrivateNetworkEndpointResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deletePrivateNetworkEndpoint(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete private network endpoint, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a private network endpoint")
}

func (r *PrivateNetworkEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: environment_id:private_network_id:service_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_network_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[2])...)
}
