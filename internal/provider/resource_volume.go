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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &VolumeResource{}
var _ resource.ResourceWithImportState = &VolumeResource{}

func NewVolumeResource() resource.Resource {
	return &VolumeResource{}
}

type VolumeResource struct {
	client *graphql.Client
}

type VolumeResourceModel struct {
	Id            types.String  `tfsdk:"id"`
	Name          types.String  `tfsdk:"name"`
	ProjectId     types.String  `tfsdk:"project_id"`
	ServiceId     types.String  `tfsdk:"service_id"`
	EnvironmentId types.String  `tfsdk:"environment_id"`
	MountPath     types.String  `tfsdk:"mount_path"`
	SizeMB        types.Float64 `tfsdk:"size_mb"`
}

func (r *VolumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *VolumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway volume. Creates a persistent volume attached to a service in a specific environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the volume.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the volume.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the volume belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the volume is attached to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the volume belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"mount_path": schema.StringAttribute{
				MarkdownDescription: "Mount path of the volume in the container.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"size_mb": schema.Float64Attribute{
				MarkdownDescription: "Size of the volume in MB.",
				Computed:            true,
				PlanModifiers: []planmodifier.Float64{
					float64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VolumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *VolumeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	environmentId := data.EnvironmentId.ValueString()

	input := VolumeCreateInput{
		MountPath:     data.MountPath.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: &environmentId,
	}

	response, err := createVolume(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume (project_id=%s, service_id=%s), got error: %s", data.ProjectId.ValueString(), data.ServiceId.ValueString(), err))
		return
	}

	tflog.Trace(ctx, "created a volume")

	volume := response.VolumeCreate.Volume

	data.Id = types.StringValue(volume.Id)

	// Save the planned name before overwriting with server default
	plannedName := data.Name

	data.Name = types.StringValue(volume.Name)

	// Save state immediately so Terraform tracks this resource.
	// If the name update or readback fails, the resource will be tainted
	// and scheduled for destroy+recreate on the next apply.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update name if the user specified one that differs from the server default
	if !plannedName.IsNull() && !plannedName.IsUnknown() && plannedName.ValueString() != volume.Name {
		_, err = updateVolume(ctx, *r.client, volume.Id, VolumeUpdateInput{
			Name: plannedName.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume name (id=%s, project_id=%s, service_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), data.ServiceId.ValueString(), err))
			return
		}

		tflog.Trace(ctx, "updated volume name")
	}

	// Read back to get the final state
	err = r.readVolumeState(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume after creation (id=%s, project_id=%s, service_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), data.ServiceId.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *VolumeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readVolumeState(ctx, data)

	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume (id=%s, project_id=%s, service_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), data.ServiceId.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *VolumeResourceModel
	var state *VolumeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update name if changed
	if data.Name.ValueString() != state.Name.ValueString() {
		_, err := updateVolume(ctx, *r.client, state.Id.ValueString(), VolumeUpdateInput{
			Name: data.Name.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume name, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "updated volume name")
	}

	// Update mount path if changed
	if data.MountPath.ValueString() != state.MountPath.ValueString() {
		_, err := updateVolumeInstance(ctx, *r.client, state.Id.ValueString(), VolumeInstanceUpdateInput{
			MountPath: data.MountPath.ValueString(),
			ServiceId: data.ServiceId.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume mount path, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "updated volume mount path")
	}

	// Read back to get the final state
	data.Id = state.Id
	err := r.readVolumeState(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume after update, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *VolumeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteVolume(ctx, *r.client, data.Id.ValueString())

	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a volume")
}

func (r *VolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:volume_id:service_id:environment_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[3])...)
}

// readVolumeState queries the project's volumes and populates the model with matching volume data.
func (r *VolumeResource) readVolumeState(ctx context.Context, data *VolumeResourceModel) error {
	response, err := getVolumeInstances(ctx, *r.client, data.ProjectId.ValueString())

	if err != nil {
		return err
	}

	for _, volume := range response.Project.Volumes.Edges {
		if volume.Node.Id != data.Id.ValueString() {
			continue
		}

		data.Name = types.StringValue(volume.Node.Name)

		for _, instance := range volume.Node.VolumeInstances.Edges {
			// Match by environment and service
			matchesEnvironment := data.EnvironmentId.IsNull() || data.EnvironmentId.IsUnknown() || instance.Node.EnvironmentId == data.EnvironmentId.ValueString()
			matchesService := data.ServiceId.IsNull() || data.ServiceId.IsUnknown() || instance.Node.ServiceId == data.ServiceId.ValueString()

			if matchesEnvironment && matchesService {
				data.EnvironmentId = types.StringValue(instance.Node.EnvironmentId)
				data.ServiceId = types.StringValue(instance.Node.ServiceId)
				data.MountPath = types.StringValue(instance.Node.MountPath)
				data.SizeMB = types.Float64Value(float64(instance.Node.SizeMB))
				return nil
			}
		}

		return &NotFoundError{ResourceType: "volume instance", Id: data.Id.ValueString()}
	}

	return &NotFoundError{ResourceType: "volume", Id: data.Id.ValueString()}
}
