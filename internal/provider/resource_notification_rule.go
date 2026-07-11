package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &NotificationRuleResource{}
var _ resource.ResourceWithImportState = &NotificationRuleResource{}

func NewNotificationRuleResource() resource.Resource {
	return &NotificationRuleResource{}
}

type NotificationRuleResource struct {
	client *graphql.Client
}

type NotificationRuleResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	WorkspaceId           types.String `tfsdk:"workspace_id"`
	ProjectId             types.String `tfsdk:"project_id"`
	EventTypes            types.List   `tfsdk:"event_types"`
	Severities            types.List   `tfsdk:"severities"`
	EphemeralEnvironments types.Bool   `tfsdk:"ephemeral_environments"`
	ChannelConfigs        types.List   `tfsdk:"channel_configs"`
}

func (r *NotificationRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_rule"
}

func (r *NotificationRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Railway notification rule. Sends notifications (via webhook, Slack, email, or other channels) when events of the configured types occur. Replaces the deprecated `railway_webhook` resource — webhooks are now one channel type among many.",
		Description:         "Railway notification rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the notification rule.",
				Description:         "Identifier of the notification rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace the rule belongs to.",
				Description:         "Identifier of the workspace the rule belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the rule scopes to. When omitted, the rule applies to the entire workspace.",
				Description:         "Identifier of the project the rule scopes to. When omitted, the rule applies to the entire workspace.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"event_types": schema.ListAttribute{
				MarkdownDescription: "Event types that trigger the rule (e.g. `deployment.completed`, `deployment.failed`).",
				Description:         "Event types that trigger the rule.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"severities": schema.ListAttribute{
				MarkdownDescription: "Severity levels to notify on. Each value must be one of `CRITICAL`, `INFO`, `NOTICE`, `WARNING`.",
				Description:         "Severity levels to notify on. Each value must be one of CRITICAL, INFO, NOTICE, WARNING.",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("CRITICAL", "INFO", "NOTICE", "WARNING"),
					),
				},
			},
			"ephemeral_environments": schema.BoolAttribute{
				MarkdownDescription: "Whether to notify for events on ephemeral (PR) environments.",
				Description:         "Whether to notify for events on ephemeral environments.",
				Optional:            true,
			},
			"channel_configs": schema.ListAttribute{
				MarkdownDescription: "List of channel configurations as JSON strings. Each entry describes one delivery channel (webhook, Slack, email, etc.). Refer to the Railway API documentation for the supported channel shapes. Example: `jsonencode({type = \"webhook\", url = \"https://example.com/hook\"})`.",
				Description:         "List of channel configurations as JSON strings.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

func (r *NotificationRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := providerDataFrom(req.ProviderData, &resp.Diagnostics)
	if data == nil {
		return
	}

	r.client = data.Client
}

func (r *NotificationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *NotificationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelConfigs, ds := decodeChannelConfigs(ctx, data.ChannelConfigs)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}

	eventTypes, ds := stringListToSlice(ctx, data.EventTypes)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}

	severities, ds := stringListToSeverities(ctx, data.Severities)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := CreateNotificationRuleInput{
		WorkspaceId:    data.WorkspaceId.ValueString(),
		EventTypes:     eventTypes,
		Severities:     severities,
		ChannelConfigs: channelConfigs,
	}
	if !data.EphemeralEnvironments.IsNull() && !data.EphemeralEnvironments.IsUnknown() {
		v := data.EphemeralEnvironments.ValueBool()
		input.EphemeralEnvironments = &v
	}
	if !data.ProjectId.IsNull() && !data.ProjectId.IsUnknown() {
		v := data.ProjectId.ValueString()
		input.ProjectId = &v
	}

	response, err := createNotificationRule(ctx, *r.client, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create notification rule (workspace_id=%s), got error: %s", data.WorkspaceId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created a notification rule")

	rule := response.NotificationRuleCreate.NotificationRuleFields
	data.Id = types.StringValue(rule.Id)
	if rule.ProjectId != "" {
		data.ProjectId = types.StringValue(rule.ProjectId)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := ""
	if !data.ProjectId.IsNull() {
		projectId = data.ProjectId.ValueString()
	}
	response, err := getNotificationRules(ctx, *r.client, data.WorkspaceId.ValueString(), projectId)

	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification rules (workspace_id=%s), got error: %s", data.WorkspaceId.ValueString(), err))
		return
	}

	var found bool
	for _, rule := range response.NotificationRules {
		if rule.Id == data.Id.ValueString() {
			if rule.ProjectId != "" {
				data.ProjectId = types.StringValue(rule.ProjectId)
			}
			// Preserve null if user didn't set ephemeral_environments and API returned the default (false).
			if !data.EphemeralEnvironments.IsNull() || rule.EphemeralEnvironments {
				data.EphemeralEnvironments = types.BoolValue(rule.EphemeralEnvironments)
			}

			eventTypeValues, d := types.ListValueFrom(ctx, types.StringType, rule.EventTypes)
			resp.Diagnostics.Append(d...)
			data.EventTypes = eventTypeValues

			sevStrs := make([]string, len(rule.Severities))
			for i, s := range rule.Severities {
				sevStrs[i] = string(s)
			}
			if data.Severities.IsNull() && len(sevStrs) == 0 {
				// preserve null
			} else {
				sevValues, d := types.ListValueFrom(ctx, types.StringType, sevStrs)
				resp.Diagnostics.Append(d...)
				data.Severities = sevValues
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

func (r *NotificationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *NotificationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelConfigs, ds := decodeChannelConfigs(ctx, data.ChannelConfigs)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}

	eventTypes, ds := stringListToSlice(ctx, data.EventTypes)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}

	severities, ds := stringListToSeverities(ctx, data.Severities)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := UpdateNotificationRuleInput{
		EventTypes:     eventTypes,
		Severities:     severities,
		ChannelConfigs: channelConfigs,
	}
	if !data.EphemeralEnvironments.IsNull() && !data.EphemeralEnvironments.IsUnknown() {
		v := data.EphemeralEnvironments.ValueBool()
		input.EphemeralEnvironments = &v
	}

	_, err := updateNotificationRule(ctx, *r.client, data.Id.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update notification rule (id=%s), got error: %s", data.Id.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "updated a notification rule")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteNotificationRule(ctx, *r.client, data.Id.ValueString())
	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification rule (id=%s), got error: %s", data.Id.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "deleted a notification rule")
}

func (r *NotificationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_id:rule_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// decodeChannelConfigs parses the user-supplied list of JSON strings into json.RawMessage values.
func decodeChannelConfigs(ctx context.Context, list types.List) ([]json.RawMessage, diag.Diagnostics) {
	var configs []string
	d := list.ElementsAs(ctx, &configs, false)
	if d.HasError() {
		return nil, d
	}

	result := make([]json.RawMessage, 0, len(configs))
	for i, s := range configs {
		if !json.Valid([]byte(s)) {
			d.AddAttributeError(path.Root("channel_configs"), "Invalid JSON", fmt.Sprintf("channel_configs[%d] is not valid JSON: %s", i, s))
			return nil, d
		}
		result = append(result, json.RawMessage(s))
	}
	return result, d
}

func stringListToSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var out []string
	d := list.ElementsAs(ctx, &out, false)
	return out, d
}

func stringListToSeverities(ctx context.Context, list types.List) ([]NotificationSeverity, diag.Diagnostics) {
	var strs []string
	d := list.ElementsAs(ctx, &strs, false)
	if d.HasError() {
		return nil, d
	}
	out := make([]NotificationSeverity, len(strs))
	for i, s := range strs {
		out[i] = NotificationSeverity(s)
	}
	return out, d
}
