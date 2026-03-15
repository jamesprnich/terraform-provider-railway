package provider

import (
	"context"
	"fmt"
	"regexp"

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

var webhookURLRegexp = regexp.MustCompile(`^https?://`)

var _ resource.Resource = &WebhookResource{}
var _ resource.ResourceWithImportState = &WebhookResource{}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

type WebhookResource struct {
	client *graphql.Client
}

type WebhookResourceModel struct {
	Id        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Url       types.String `tfsdk:"url"`
	Filters   types.List   `tfsdk:"filters"`
}

func (r *WebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway webhook.",
		Description:         "Railway webhook.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the webhook.",
				Description:         "Identifier of the webhook.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the webhook belongs to.",
				Description:         "Identifier of the project the webhook belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "URL of the webhook.",
				Description:         "URL of the webhook.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.RegexMatches(webhookURLRegexp, "must be a valid HTTP or HTTPS URL"),
				},
			},
			"filters": schema.ListAttribute{
				MarkdownDescription: "List of event filters for the webhook.",
				Description:         "List of event filters for the webhook.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *WebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := WebhookCreateInput{
		ProjectId: data.ProjectId.ValueString(),
		Url:       data.Url.ValueString(),
	}

	if !data.Filters.IsNull() && !data.Filters.IsUnknown() {
		var filters []string

		resp.Diagnostics.Append(data.Filters.ElementsAs(ctx, &filters, false)...)

		if resp.Diagnostics.HasError() {
			return
		}

		input.Filters = filters
	}

	response, err := createWebhook(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create webhook (project_id=%s), got error: %s", data.ProjectId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created a webhook")

	webhook := response.WebhookCreate.ProjectWebhook

	data.Id = types.StringValue(webhook.Id)
	data.ProjectId = types.StringValue(webhook.ProjectId)
	data.Url = types.StringValue(webhook.Url)

	// Preserve null if filters was not specified in config
	if !data.Filters.IsNull() {
		filtersValue, diags := types.ListValueFrom(ctx, types.StringType, webhook.Filters)
		resp.Diagnostics.Append(diags...)

		if resp.Diagnostics.HasError() {
			return
		}

		data.Filters = filtersValue
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *WebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getWebhooks(ctx, *r.client, data.ProjectId.ValueString())

	if isNotFound(err) {
		tflog.Info(ctx, "webhooks not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read webhooks (id=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), err))
		return
	}

	var found bool

	for _, edge := range response.Webhooks.Edges {
		webhook := edge.Node.ProjectWebhook

		if webhook.Id == data.Id.ValueString() {
			data.Id = types.StringValue(webhook.Id)
			data.ProjectId = types.StringValue(webhook.ProjectId)
			data.Url = types.StringValue(webhook.Url)

			// Preserve null when API returns empty and state was null (user didn't specify filters)
			if data.Filters.IsNull() && len(webhook.Filters) == 0 {
				// Keep null — don't convert to empty list
			} else {
				filtersValue, diags := types.ListValueFrom(ctx, types.StringType, webhook.Filters)
				resp.Diagnostics.Append(diags...)

				if resp.Diagnostics.HasError() {
					return
				}

				data.Filters = filtersValue
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

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := WebhookUpdateInput{
		Url: data.Url.ValueString(),
	}

	if !data.Filters.IsNull() && !data.Filters.IsUnknown() {
		var filters []string

		resp.Diagnostics.Append(data.Filters.ElementsAs(ctx, &filters, false)...)

		if resp.Diagnostics.HasError() {
			return
		}

		input.Filters = filters
	}

	response, err := updateWebhook(ctx, *r.client, data.Id.ValueString(), input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update webhook (id=%s), got error: %s", data.Id.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "updated a webhook")

	webhook := response.WebhookUpdate.ProjectWebhook

	data.Id = types.StringValue(webhook.Id)
	data.ProjectId = types.StringValue(webhook.ProjectId)
	data.Url = types.StringValue(webhook.Url)

	// Preserve null when API returns empty and config didn't specify filters
	if data.Filters.IsNull() && len(webhook.Filters) == 0 {
		// Keep null
	} else {
		filtersValue, diags := types.ListValueFrom(ctx, types.StringType, webhook.Filters)
		resp.Diagnostics.Append(diags...)

		if resp.Diagnostics.HasError() {
			return
		}

		data.Filters = filtersValue
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *WebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteWebhook(ctx, *r.client, data.Id.ValueString())

	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete webhook (id=%s), got error: %s", data.Id.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "deleted a webhook")
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:webhook_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
