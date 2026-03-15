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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &EgressGatewayResource{}
var _ resource.ResourceWithImportState = &EgressGatewayResource{}

func NewEgressGatewayResource() resource.Resource {
	return &EgressGatewayResource{}
}

type EgressGatewayResource struct {
	client *graphql.Client
}

type EgressGatewayResourceModel struct {
	Id            types.String `tfsdk:"id"`
	ServiceId     types.String `tfsdk:"service_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	Region        types.String `tfsdk:"region"`
	IpAddresses   types.List   `tfsdk:"ip_addresses"`
}

func (r *EgressGatewayResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_egress_gateway"
}

func (r *EgressGatewayResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway egress gateway. Associates a static egress IP with a service in a specific environment.",
		Description:         "Railway egress gateway. Associates a static egress IP with a service in a specific environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the egress gateway (service_id:environment_id).",
				Description:         "Identifier of the egress gateway (service_id:environment_id).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the egress gateway belongs to.",
				Description:         "Identifier of the service the egress gateway belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the egress gateway belongs to.",
				Description:         "Identifier of the environment the egress gateway belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region for the egress gateway.",
				Description:         "Region for the egress gateway.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip_addresses": schema.ListAttribute{
				MarkdownDescription: "List of static IPv4 addresses assigned to the egress gateway.",
				Description:         "List of static IPv4 addresses assigned to the egress gateway.",
				ElementType:         types.StringType,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *EgressGatewayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EgressGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *EgressGatewayResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := EgressGatewayCreateInput{
		ServiceId:     data.ServiceId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
	}

	if !data.Region.IsNull() && !data.Region.IsUnknown() {
		region := data.Region.ValueString()
		input.Region = &region
	}

	response, err := createEgressGateway(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create egress gateway (service_id=%s, environment_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created an egress gateway")

	// Save state immediately so Terraform can track (and destroy) the resource
	// even if the process crashes during IP list processing. Resolve Unknown
	// computed values to concrete values before saving.
	data.Id = types.StringValue(fmt.Sprintf("%s:%s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString()))
	if data.IpAddresses.IsUnknown() {
		data.IpAddresses = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipAddresses := make([]string, len(response.EgressGatewayAssociationCreate))
	for i, gw := range response.EgressGatewayAssociationCreate {
		ipAddresses[i] = gw.Ipv4
	}

	ipList, diags := types.ListValueFrom(ctx, types.StringType, ipAddresses)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.IpAddresses = ipList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EgressGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *EgressGatewayResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getEgressGateways(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if isNotFound(err) {
		tflog.Info(ctx, "egress gateways not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read egress gateways (service_id=%s, environment_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), err))
		return
	}

	if len(response.EgressGateways) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	ipAddresses := make([]string, len(response.EgressGateways))
	for i, gw := range response.EgressGateways {
		ipAddresses[i] = gw.Ipv4
	}

	ipList, diags := types.ListValueFrom(ctx, types.StringType, ipAddresses)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.IpAddresses = ipList

	// Only persist region if the user specified it in config (already in state).
	// Region has RequiresReplace — setting it when the user didn't specify it
	// would cause a spurious destroy+recreate on the next plan.
	if !data.Region.IsNull() && len(response.EgressGateways) > 0 && response.EgressGateways[0].Region != "" {
		data.Region = types.StringValue(response.EgressGateways[0].Region)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EgressGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *EgressGatewayResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EgressGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *EgressGatewayResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := EgressGatewayServiceTargetInput{
		ServiceId:     data.ServiceId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
	}

	_, err := clearEgressGateways(ctx, *r.client, input)

	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to clear egress gateways (service_id=%s, environment_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "deleted egress gateways")
}

func (r *EgressGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
