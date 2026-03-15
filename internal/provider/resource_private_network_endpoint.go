package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
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
	Id               types.String   `tfsdk:"id"`
	PrivateNetworkId types.String   `tfsdk:"private_network_id"`
	ServiceId        types.String   `tfsdk:"service_id"`
	EnvironmentId    types.String   `tfsdk:"environment_id"`
	ServiceName      types.String   `tfsdk:"service_name"`
	DnsName          types.String   `tfsdk:"dns_name"`
	PrivateIps       types.List     `tfsdk:"private_ips"`
	Tags             types.List     `tfsdk:"tags"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

func (r *PrivateNetworkEndpointResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network_endpoint"
}

func (r *PrivateNetworkEndpointResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway private network endpoint. Connects a service to a private network, enabling internal service-to-service communication.",
		Description:         "Railway private network endpoint. Connects a service to a private network, enabling internal service-to-service communication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the private network endpoint (publicId).",
				Description:         "Identifier of the private network endpoint (publicId).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_network_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the private network to connect to.",
				Description:         "Identifier of the private network to connect to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service to connect.",
				Description:         "Identifier of the service to connect.",
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
				Description:         "Identifier of the environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_name": schema.StringAttribute{
				MarkdownDescription: "Name of the service. Required for creation (used in the create API input), but not needed for import since it is not returned by the API.",
				Description:         "Name of the service. Required for creation (used in the create API input), but not needed for import since it is not returned by the API.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "DNS name of the endpoint within the private network. Can be changed via rename.",
				Description:         "DNS name of the endpoint within the private network. Can be changed via rename.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_ips": schema.ListAttribute{
				MarkdownDescription: "List of private IP addresses assigned to the endpoint.",
				Description:         "List of private IP addresses assigned to the endpoint.",
				ElementType:         types.StringType,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.ListAttribute{
				MarkdownDescription: "Tags associated with the private network endpoint.",
				Description:         "Tags associated with the private network endpoint.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
			}),
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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create private network endpoint (service_id=%s, environment_id=%s, private_network_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created a private network endpoint")

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
	// If the rename or consistency wait fails, the resource will be tainted
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
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to rename private network endpoint (service_id=%s, environment_id=%s, private_network_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), err))
				return
			}

			data.DnsName = planDnsName

			tflog.Debug(ctx, "renamed private network endpoint")
		}
	}

	// Consistency waiter: poll the GET endpoint until it returns valid data
	// on 2 consecutive reads. This ensures Terraform's post-create refresh
	// hits a warm, consistent GET and won't produce "plan not empty."
	// Following the AWS/GCP/Azure pattern: retries belong in Create, not Read.
	createTimeout, diags := data.Timeouts.Create(ctx, 2*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	consecutiveSuccesses := 0
	waitErr := retry.RetryContext(ctx, createTimeout, func() *retry.RetryError {
		readResp, readErr := getPrivateNetworkEndpoint(ctx, *r.client, data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), data.ServiceId.ValueString())

		if readErr != nil {
			consecutiveSuccesses = 0
			return retry.RetryableError(fmt.Errorf("GET endpoint returned error: %w", readErr))
		}

		if readResp.PrivateNetworkEndpoint.PrivateNetworkEndpointFields.PublicId == "" {
			consecutiveSuccesses = 0
			return retry.RetryableError(fmt.Errorf("GET endpoint returned empty data, still propagating"))
		}

		consecutiveSuccesses++
		if consecutiveSuccesses < 2 {
			return retry.RetryableError(fmt.Errorf("need %d more consecutive successful reads", 2-consecutiveSuccesses))
		}

		return nil
	})

	if waitErr != nil {
		tflog.Warn(ctx, "GET endpoint did not stabilize within timeout — state was saved from the create mutation response, proceeding", map[string]interface{}{
			"error": waitErr.Error(),
		})
	} else {
		tflog.Trace(ctx, "GET endpoint confirmed consistent after create")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateNetworkEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *PrivateNetworkEndpointResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Railway's GET endpoint has severe eventual consistency issues — it
	// returns empty data (not a 404 error) for both "resource exists but
	// query is slow" and "resource was actually deleted." Under load the
	// GET can return null for 5+ minutes for existing resources.
	//
	// Unlike AWS/GCP/Azure APIs which return proper 404s for deleted
	// resources, Railway makes it impossible to distinguish consistency
	// lag from actual deletion. Strategy:
	//
	// - Existing resource (has id in state): single GET. If empty,
	//   preserve state — the consistency waiter in Create already
	//   verified the resource exists. If the resource was truly deleted
	//   externally, Terraform's next destroy will handle it.
	// - Import (no id in state): retry with backoff since we need the
	//   API data to populate state.
	//
	// Deletion detection relies on the Delete method and isNotFound
	// errors, not on Read returning empty.
	hasExistingId := !data.Id.IsNull() && !data.Id.IsUnknown() && data.Id.ValueString() != ""

	var endpoint getPrivateNetworkEndpointPrivateNetworkEndpoint

	if hasExistingId {
		response, err := getPrivateNetworkEndpoint(ctx, *r.client, data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), data.ServiceId.ValueString())

		if isNotFound(err) {
			tflog.Info(ctx, "private network endpoint not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network endpoint (service_id=%s, environment_id=%s, private_network_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), err))
			return
		}

		if response.PrivateNetworkEndpoint.PrivateNetworkEndpointFields.PublicId == "" {
			tflog.Warn(ctx, "private network endpoint query returned empty data (eventual consistency), preserving existing state")
			return
		}

		endpoint = response.PrivateNetworkEndpoint
	} else {
		readTimeout, diags := data.Timeouts.Read(ctx, 30*time.Second)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		err := retry.RetryContext(ctx, readTimeout, func() *retry.RetryError {
			response, readErr := getPrivateNetworkEndpoint(ctx, *r.client, data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), data.ServiceId.ValueString())

			if isNotFound(readErr) {
				return retry.NonRetryableError(readErr)
			}

			if readErr != nil {
				return retry.RetryableError(readErr)
			}

			if response.PrivateNetworkEndpoint.PrivateNetworkEndpointFields.PublicId == "" {
				return retry.RetryableError(fmt.Errorf("endpoint returned with empty PublicId, still propagating"))
			}

			endpoint = response.PrivateNetworkEndpoint
			return nil
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read private network endpoint during import (service_id=%s, environment_id=%s, private_network_id=%s), got error: %s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), data.PrivateNetworkId.ValueString(), err))
			return
		}
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

		tflog.Debug(ctx, "renamed private network endpoint")
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

	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete private network endpoint, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "deleted a private network endpoint")
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
