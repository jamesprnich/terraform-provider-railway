package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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

var _ resource.Resource = &VolumeBackupScheduleResource{}
var _ resource.ResourceWithImportState = &VolumeBackupScheduleResource{}

func NewVolumeBackupScheduleResource() resource.Resource {
	return &VolumeBackupScheduleResource{}
}

type VolumeBackupScheduleResource struct {
	client *graphql.Client
}

type VolumeBackupScheduleResourceModel struct {
	Id               types.String `tfsdk:"id"`
	VolumeInstanceId types.String `tfsdk:"volume_instance_id"`
	Kinds            types.List   `tfsdk:"kinds"`
}

func (r *VolumeBackupScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup_schedule"
}

func (r *VolumeBackupScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway volume backup schedule. Manages the backup schedule for a volume instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the volume backup schedule (same as volume_instance_id).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"volume_instance_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the volume instance.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"kinds": schema.ListAttribute{
				MarkdownDescription: "List of backup schedule kinds to enable. Valid values: `DAILY`, `WEEKLY`, `MONTHLY`.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("DAILY", "WEEKLY", "MONTHLY"),
					),
				},
			},
		},
	}
}

func (r *VolumeBackupScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeBackupScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *VolumeBackupScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	kinds, err := r.extractKinds(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse kinds, got error: %s", err))
		return
	}

	_, err = updateVolumeInstanceBackupSchedule(ctx, *r.client, kinds, data.VolumeInstanceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume backup schedule (volume_instance_id=%s), got error: %s", data.VolumeInstanceId.ValueString(), err))
		return
	}

	tflog.Trace(ctx, "created volume backup schedule")

	data.Id = data.VolumeInstanceId

	// Read back to verify
	err = r.readScheduleState(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume backup schedule after creation (volume_instance_id=%s), got error: %s", data.VolumeInstanceId.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeBackupScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *VolumeBackupScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readScheduleState(ctx, data)

	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume backup schedule (volume_instance_id=%s), got error: %s", data.VolumeInstanceId.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeBackupScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *VolumeBackupScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	kinds, err := r.extractKinds(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse kinds, got error: %s", err))
		return
	}

	_, err = updateVolumeInstanceBackupSchedule(ctx, *r.client, kinds, data.VolumeInstanceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume backup schedule, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated volume backup schedule")

	// Read back to verify
	err = r.readScheduleState(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume backup schedule after update, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeBackupScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *VolumeBackupScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete by setting an empty kinds list
	_, err := updateVolumeInstanceBackupSchedule(ctx, *r.client, []VolumeInstanceBackupScheduleKind{}, data.VolumeInstanceId.ValueString())

	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume backup schedule, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted volume backup schedule")
}

func (r *VolumeBackupScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_instance_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// extractKinds converts the Terraform list of strings into a slice of VolumeInstanceBackupScheduleKind.
func (r *VolumeBackupScheduleResource) extractKinds(ctx context.Context, data *VolumeBackupScheduleResourceModel) ([]VolumeInstanceBackupScheduleKind, error) {
	var kindStrings []string

	diags := data.Kinds.ElementsAs(ctx, &kindStrings, false)

	if diags.HasError() {
		return nil, fmt.Errorf("unable to extract kinds from list")
	}

	kinds := make([]VolumeInstanceBackupScheduleKind, len(kindStrings))

	for i, k := range kindStrings {
		kinds[i] = VolumeInstanceBackupScheduleKind(k)
	}

	return kinds, nil
}

// readScheduleState queries the backup schedules and populates the model.
func (r *VolumeBackupScheduleResource) readScheduleState(ctx context.Context, data *VolumeBackupScheduleResourceModel) error {
	response, err := getVolumeInstanceBackupSchedules(ctx, *r.client, data.VolumeInstanceId.ValueString())

	if err != nil {
		return err
	}

	kindStrings := make([]string, len(response.VolumeInstanceBackupScheduleList))

	for i, schedule := range response.VolumeInstanceBackupScheduleList {
		kindStrings[i] = string(schedule.Kind)
	}

	kindValues, diags := types.ListValueFrom(ctx, types.StringType, kindStrings)

	if diags.HasError() {
		return fmt.Errorf("unable to build kinds list from API response")
	}

	data.Kinds = kindValues

	return nil
}
