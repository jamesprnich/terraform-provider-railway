package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}
var _ resource.ResourceWithModifyPlan = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

type ServiceResource struct {
	client           *graphql.Client
	strictEnvScoping bool
}

type ServiceResourceVolumeModel struct {
	Id        types.String  `tfsdk:"id"`
	Name      types.String  `tfsdk:"name"`
	MountPath types.String  `tfsdk:"mount_path"`
	Size      types.Float64 `tfsdk:"size"`
}

var volumeAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"name":       types.StringType,
	"mount_path": types.StringType,
	"size":       types.Float64Type,
}

type ServiceResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	Icon          types.String `tfsdk:"icon"`
	Volume        types.Object `tfsdk:"volume"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 2,
		MarkdownDescription: "Railway service — an empty per-environment shell. Under `strict_env_scoping = true` " +
			"(provider default), `environment_id` is required and the service is created only in that environment. " +
			"Attach source, build, and deploy configuration via `railway_service_instance` (per-environment). " +
			"Optional inline `volume` block creates a persistent volume scoped to the same environment.\n\n" +
			"> Setting source or build configuration on `railway_service` is intentionally not supported — the " +
			"underlying Railway mutation (`serviceConnect`) is env-less and would create source connections " +
			"across every non-fork environment in the project. Use `railway_service_instance` instead.",
		Description: "Railway service — an empty per-environment shell. Attach source and deploy configuration " +
			"via railway_service_instance (per-environment). Optional inline volume creates a persistent volume.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service.",
				Description:         "Identifier of the service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service. Also becomes the service's private DNS name " +
					"(`<name>.railway.internal`).\n\n" +
					"~> **Service names are unique per project, not per environment.** Railway rejects " +
					"`serviceCreate` for a name that already exists in the project regardless of which " +
					"environment the new service is scoped to. To run the same role (e.g., `backend`) in " +
					"multiple environments, prefix the name with the environment: `dev-backend`, `tst-backend`, " +
					"`prd-backend`. This convention also matches how Railway's private DNS works — the DNS " +
					"name is just the service name.",
				Description: "Name of the service. Unique per project, not per environment.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the service belongs to. ~> **Warning:** Changing this forces resource destruction and recreation.",
				Description:         "Identifier of the project the service belongs to. Warning: Changing this forces resource destruction and recreation.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the service is scoped to. Required under " +
					"strict env-scoping (provider default). Must reference a **fork environment** (an environment " +
					"created with `source_environment_id`). This is a Railway API property, not a provider " +
					"limitation: Railway's `serviceCreate` silently ignores `environmentId` when it points at a " +
					"non-fork environment and creates the service across every non-fork environment in the project. " +
					"Cannot be changed after creation.\n\n" +
					"~> **Dependency note:** `railway_service` references only `project_id`, so Terraform cannot " +
					"infer the environment dependency from the reference in `environment_id`. Add " +
					"`depends_on = [railway_environment.<name>]` on the service resource, or the environment may " +
					"not exist when `serviceCreate` runs.",
				Description: "Identifier of the environment the service is scoped to. Required under strict env-scoping.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"icon": schema.StringAttribute{
				MarkdownDescription: "Icon displayed for the service in the Railway dashboard. Cosmetic, " +
					"applies project-wide (not per-environment). See [Railway's icon docs](https://docs.railway.com/reference/services).",
				Description: "Icon displayed for the service in the Railway dashboard. Cosmetic, applies project-wide.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"volume": schema.SingleNestedAttribute{
				MarkdownDescription: "Volume connected to the service. Created in the same environment as the " +
					"service (`environment_id`). Prefer the standalone `railway_volume` resource when the " +
					"volume needs its own lifecycle, backup schedule, or references from other resources.",
				Description: "Volume connected to the service. Created in the same environment as the service.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Identifier of the volume.",
						Description:         "Identifier of the volume.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							useStringStateForUnknownIfNonNull(),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the volume.",
						Description:         "Name of the volume.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.UTF8LengthAtLeast(1),
						},
					},
					"mount_path": schema.StringAttribute{
						MarkdownDescription: "Mount path of the volume.",
						Description:         "Mount path of the volume.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.UTF8LengthAtLeast(1),
						},
					},
					"size": schema.Float64Attribute{
						MarkdownDescription: "Size of the volume in MB.",
						Description:         "Size of the volume in MB.",
						Computed:            true,
						PlanModifiers: []planmodifier.Float64{
							useFloat64StateForUnknownIfNonNull(),
						},
					},
				},
			},
		},
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := providerDataFrom(req.ProviderData, &resp.Diagnostics)
	if data == nil {
		return
	}

	r.client = data.Client
	r.strictEnvScoping = data.StrictEnvScoping
}

// ModifyPlan runs the strict-env-scoping check at plan time so the user sees
// the diagnostic on `tofu plan`, not only on `tofu apply`. It intentionally
// reads the CONFIG (not the plan) so that:
//
//   - An unset optional attribute (config = null, plan = unknown because of
//     the Computed marker) is caught as "missing environment_id".
//   - An attribute set to a computed reference (config = a traversal, plan =
//     unknown) is accepted — the value will resolve at apply time.
//
// Delete plans have a null Raw plan; we skip in that case.
func (r *ServiceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config ServiceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.strictEnvScoping && config.EnvironmentId.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("environment_id"),
			"environment_id is required under strict env-scoping",
			"The Railway provider defaults to strict env-scoping (`strict_env_scoping = true`) to prevent "+
				"services from being created project-wide. Set `environment_id` to a fork environment "+
				"(typically `railway_environment.<name>.id`) so the service is scoped to that environment "+
				"only. To opt out and create services across every non-fork environment (Railway's default), "+
				"set `strict_env_scoping = false` on the provider block — you own the leak surface.",
		)
	}
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceResourceModel
	var volumeData *ServiceResourceVolumeModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Under strict mode, environment_id must be set in config. Since the
	// attribute is Optional+Computed, an unset config surfaces here as
	// Unknown (not Null) — treat both as "the user didn't supply one."
	// ModifyPlan catches this at plan time; this is the apply-time backstop
	// for edge cases where the plan value was expected to resolve but did
	// not (e.g., a broken reference).
	if r.strictEnvScoping && (data.EnvironmentId.IsNull() || data.EnvironmentId.IsUnknown()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("environment_id"),
			"environment_id is required under strict env-scoping",
			"The Railway provider defaults to strict env-scoping (`strict_env_scoping = true`) to prevent "+
				"services from being created project-wide. Set `environment_id` to a fork environment "+
				"(typically `railway_environment.<name>.id`) so the service is scoped to that environment "+
				"only. To opt out and create services across every non-fork environment (Railway's default), "+
				"set `strict_env_scoping = false` on the provider block — you own the leak surface.",
		)
		return
	}

	// Non-fork target check (the B4 footgun): Railway's serviceCreate silently
	// ignores environmentId when the target is a non-fork environment, and
	// creates the service across every non-fork env in the project. Under
	// strict mode we look the target up and reject if it is not a fork. If
	// the lookup itself errors (network/transient), we fail open — Railway's
	// own semantics still apply, we just don't get to emit the helpful error.
	if r.strictEnvScoping && !data.EnvironmentId.IsNull() && !data.EnvironmentId.IsUnknown() {
		envId := data.EnvironmentId.ValueString()
		envsResp, envsErr := getEnvironments(ctx, *r.client, data.ProjectId.ValueString())
		if envsErr == nil {
			for _, edge := range envsResp.Environments.Edges {
				if edge.Node.Id == envId {
					if edge.Node.SourceEnvironment.Id == "" {
						resp.Diagnostics.AddAttributeError(
							path.Root("environment_id"),
							"environment_id must reference a fork environment under strict env-scoping",
							fmt.Sprintf("The target environment %q (id=%s) is not a fork — it has no "+
								"source_environment_id. Railway's `serviceCreate` silently ignores "+
								"`environmentId` when the target is a non-fork environment, and creates the "+
								"service across every non-fork environment in the project. Either declare "+
								"the target environment as a fork (set `source_environment_id`), or set "+
								"`strict_env_scoping = false` on the provider block to accept this behaviour.",
								edge.Node.Name, envId),
						)
						return
					}
					break
				}
			}
		}
	}

	// Determine which environment volumes and post-create reads should target.
	// Priority: user-supplied environment_id, then the project's default env.
	envId, err := resolveTargetEnvironment(ctx, *r.client, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to determine target environment (project_id=%s), got error: %s", data.ProjectId.ValueString(), err))
		return
	}

	name := data.Name.ValueString()
	input := ServiceCreateInput{
		Name:      &name,
		ProjectId: data.ProjectId.ValueString(),
	}
	// Same null-vs-unknown consideration as railway_environment: an unset
	// Optional+Computed attribute is Unknown, not Null, and sending an empty
	// environmentId would leak the service across every non-fork environment.
	if !data.EnvironmentId.IsNull() && !data.EnvironmentId.IsUnknown() {
		envIdVal := data.EnvironmentId.ValueString()
		input.EnvironmentId = &envIdVal
	}
	if !data.Icon.IsNull() {
		iconVal := data.Icon.ValueString()
		input.Icon = &iconVal
	}

	response, err := createService(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service %q (project_id=%s), got error: %s", data.Name.ValueString(), data.ProjectId.ValueString(), err))
		return
	}

	tflog.Debug(ctx, "created a service")

	service := response.ServiceCreate.Service

	data.Id = types.StringValue(service.Id)
	data.Name = types.StringValue(service.Name)
	data.ProjectId = types.StringValue(service.ProjectId)
	data.Icon = readOptionalString(service.Icon)
	// Only overwrite EnvironmentId in state if the user explicitly set it.
	// If unset (permissive mode with no env id), we don't fabricate a value —
	// the service lands in Railway's "all non-forks" pool and there is no
	// single environment id that represents it.
	if data.EnvironmentId.IsNull() || data.EnvironmentId.IsUnknown() {
		data.EnvironmentId = types.StringNull()
	}

	// Extract volume plan data before nulling it for the early state save.
	// The volume hasn't been created yet, so we must not store unknown
	// computed sub-fields (id, size) in state — that causes
	// "Provider returned invalid result object after apply" errors.
	hasPlannedVolume := !data.Volume.IsNull() && !data.Volume.IsUnknown()

	if hasPlannedVolume {
		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	data.Volume = types.ObjectNull(volumeAttrTypes)

	// Save state immediately so Terraform tracks this resource. If any
	// subsequent step fails, the resource will be tainted and scheduled
	// for destroy+recreate on the next apply.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if hasPlannedVolume {
		serviceId := data.Id.ValueString()

		// Retry volume creation — Railway's API may return "Problem processing
		// request" on newly created services due to eventual consistency.
		var volumeResponse *createVolumeResponse
		err = retryCreateContext(ctx, 30*time.Second, func() error {
			var createErr error
			volumeResponse, createErr = createVolume(ctx, *r.client, VolumeCreateInput{
				MountPath:     volumeData.MountPath.ValueString(),
				ProjectId:     data.ProjectId.ValueString(),
				ServiceId:     &serviceId,
				EnvironmentId: &envId,
			})
			return createErr
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume (service_id=%s, project_id=%s, environment_id=%s), got error: %s", serviceId, data.ProjectId.ValueString(), envId, err))
			return
		}

		tflog.Debug(ctx, "created a volume")

		volId := volumeResponse.VolumeCreate.Id
		desiredName := volumeData.Name.ValueString()

		// Rename only if the volume's current name genuinely differs from the
		// desired name. Railway auto-names newly-created volumes as
		// {service-name}-volume; the read-then-rename path safely handles both
		// the differ-and-rename and no-op cases without incurring the rejected
		// self-collision that would fire if we always issued updateVolume.
		if err := renameServiceVolumeIfNeeded(ctx, *r.client, data.ProjectId.ValueString(), volId, desiredName); err != nil {
			if _, delErr := deleteVolume(ctx, *r.client, volId); delErr != nil {
				tflog.Warn(ctx, fmt.Sprintf("failed to clean up orphaned volume %s: %s", volId, delErr))
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to rename volume (service_id=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), err))
			return
		}
	}

	// Retry the volume readback across Railway's post-create eventual-
	// consistency window. Without this, a service that was just given an
	// inline volume can have data.Volume stay null in state (the volume
	// instance hasn't appeared in getVolumeInstances yet), and Terraform
	// rejects the create with "inconsistent result after apply". Empirically
	// this window is usually 2–5 s but has been observed exceeding 28 s
	// during workspace-wide slow moments — a generous 90 s budget preserves
	// success in the tail without hanging the apply on a real failure.
	err = retryReadAfterCreateContext(ctx, 90*time.Second, func() error {
		if buildErr := getAndBuildVolumeInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), envId, data); buildErr != nil {
			return buildErr
		}
		if hasPlannedVolume && data.Volume.IsNull() {
			// Wrap a NotFoundError sentinel so retryReadAfterCreateContext
			// classifies this as retryable — the volumeInstances edge is
			// populated a few seconds after createVolume returns, and without
			// the sentinel the retry loop misclassifies the "not yet visible"
			// signal as terminal and bails out inside a single poll interval.
			return fmt.Errorf("inline volume not yet visible for service %s in environment %s: %w",
				data.Id.ValueString(), envId,
				&NotFoundError{ResourceType: "inline volume for service", Id: data.Id.ValueString()})
		}
		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume settings (service_id=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getService(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service (id=%s, name=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.Name.ValueString(), data.ProjectId.ValueString(), err))
		return
	}

	service := response.Service.Service

	data.Id = types.StringValue(service.Id)
	data.Name = types.StringValue(service.Name)
	data.ProjectId = types.StringValue(service.ProjectId)
	data.Icon = readOptionalString(service.Icon)

	// Only refresh the inline volume from the API if state already claims
	// one. Otherwise a standalone `railway_volume` targeting the same
	// service+env would be misread as an inline volume and Terraform would
	// plan to remove it on every subsequent apply.
	if !data.Volume.IsNull() {
		envId, err := resolveTargetEnvironment(ctx, *r.client, data)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to determine target environment for readback (service_id=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), err))
			return
		}

		err = getAndBuildVolumeInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), envId, data)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume settings (service_id=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceResourceModel
	var volumeData *ServiceResourceVolumeModel

	var state *ServiceResourceModel
	var volumeState *ServiceResourceVolumeModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.ValueString() != state.Name.ValueString() || !data.Icon.Equal(state.Icon) {
		input := ServiceUpdateInput{
			Name: data.Name.ValueString(),
			Icon: data.Icon.ValueString(),
		}

		response, err := updateService(ctx, *r.client, data.Id.ValueString(), input)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service (id=%s), got error: %s", data.Id.ValueString(), err))
			return
		}

		tflog.Debug(ctx, "updated a service")

		service := response.ServiceUpdate.Service

		data.Id = types.StringValue(service.Id)
		data.Name = types.StringValue(service.Name)
		data.ProjectId = types.StringValue(service.ProjectId)
		data.Icon = readOptionalString(service.Icon)
	}

	envId, err := resolveTargetEnvironment(ctx, *r.client, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to determine target environment (service_id=%s, project_id=%s), got error: %s", data.Id.ValueString(), data.ProjectId.ValueString(), err))
		return
	}

	// Volume lifecycle within Update — delete, create, mutate — all use the
	// same env id resolved above.

	// Delete volume if it was removed from the plan.
	volumeDeleted := false
	if data.Volume.IsNull() && !state.Volume.IsNull() {
		resp.Diagnostics.Append(state.Volume.As(ctx, &volumeState, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		_, err := deleteVolume(ctx, *r.client, volumeState.Id.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume, got error: %s", err))
			return
		}

		volumeDeleted = true
		tflog.Debug(ctx, "deleted a volume")
	}

	// Create volume if it was added
	if !data.Volume.IsNull() && state.Volume.IsNull() {
		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		serviceId := data.Id.ValueString()

		// Retry volume creation — Railway's API may return "Problem processing
		// request" due to eventual consistency.
		var volumeResponse *createVolumeResponse
		err := retryCreateContext(ctx, 30*time.Second, func() error {
			var createErr error
			volumeResponse, createErr = createVolume(ctx, *r.client, VolumeCreateInput{
				MountPath:     volumeData.MountPath.ValueString(),
				ProjectId:     data.ProjectId.ValueString(),
				ServiceId:     &serviceId,
				EnvironmentId: &envId,
			})
			return createErr
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume, got error: %s", err))
			return
		}

		tflog.Debug(ctx, "created a volume")

		volId := volumeResponse.VolumeCreate.Id
		desiredName := volumeData.Name.ValueString()

		if err := renameServiceVolumeIfNeeded(ctx, *r.client, data.ProjectId.ValueString(), volId, desiredName); err != nil {
			if _, delErr := deleteVolume(ctx, *r.client, volId); delErr != nil {
				tflog.Warn(ctx, fmt.Sprintf("failed to clean up orphaned volume %s: %s", volId, delErr))
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to rename volume, got error: %s", err))
			return
		}
	}

	// Update volume if it was changed
	if !data.Volume.IsNull() && !state.Volume.IsNull() {
		resp.Diagnostics.Append(state.Volume.As(ctx, &volumeState, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		if volumeState.Name != volumeData.Name {
			_, err := updateVolume(ctx, *r.client, volumeState.Id.ValueString(), VolumeUpdateInput{
				Name: volumeData.Name.ValueString(),
			})

			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume, got error: %s", err))
				return
			}

			tflog.Debug(ctx, "updated a volume")
		}

		if volumeState.MountPath != volumeData.MountPath {
			_, err := updateVolumeInstance(ctx, *r.client, volumeState.Id.ValueString(), VolumeInstanceUpdateInput{
				MountPath: volumeData.MountPath.ValueString(),
				ServiceId: data.Id.ValueString(),
			})

			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume instance, got error: %s", err))
				return
			}

			tflog.Debug(ctx, "updated a volume instance")
		}
	}

	// Skip volume readback if we just deleted it. Railway's volumeDelete API
	// retains volumes for data protection, so the readback would find the volume
	// still present and set data.Volume to non-null, contradicting the plan.
	if !volumeDeleted {
		err = getAndBuildVolumeInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), envId, data)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume settings, got error: %s", err))
			return
		}
	} else {
		data.Volume = types.ObjectNull(volumeAttrTypes)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ServiceResourceModel
	var volumeData *ServiceResourceVolumeModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Pass environmentId to serviceDelete so a fork-scoped service is removed
	// only from that fork. Without it, Railway's env-less delete path removes
	// the service from every non-fork environment — undesirable when a
	// permissive-mode consumer has services in multiple non-fork envs, and
	// meaningless when we scoped the service to a fork on Create.
	var envIdPtr *string
	if !data.EnvironmentId.IsNull() && !data.EnvironmentId.IsUnknown() {
		envIdVal := data.EnvironmentId.ValueString()
		envIdPtr = &envIdVal
	}

	_, err := deleteService(ctx, *r.client, data.Id.ValueString(), envIdPtr)

	if err != nil && !isNotFoundOrGone(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service, got error: %s", err))
		return
	}

	tflog.Debug(ctx, "deleted a service")

	if !data.Volume.IsNull() {
		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		_, err := deleteVolume(ctx, *r.client, volumeData.Id.ValueString())

		if err != nil && !isNotFoundOrGone(err) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume, got error: %s", err))
			return
		}

		tflog.Debug(ctx, "deleted a volume")
	}
}

// parseServiceImportId splits a railway_service import id into service_id and
// optional environment_id. Two accepted forms:
//   - `<service_id>` (bare) — a project-wide service; permitted only when
//     strict is false. Returns envId as the empty string.
//   - `<service_id>:<environment_id>` — a fork-scoped service.
//
// Errors:
//   - Bare form under strict env-scoping: fork-scoped services need
//     environment_id in state up front; without it the next plan would see
//     the fork env_id in HCL as a change requiring replace and silently
//     destroy the just-imported service.
//   - Empty service_id or empty environment_id after the colon: malformed.
//
// The function returns two error slots so callers can render "malformed id"
// and "strict-mode omission" as separate diagnostics with different
// summaries. Pure — safe to unit-test without the framework.
type serviceImportIdError struct {
	summary string
	detail  string
}

func parseServiceImportId(raw string, strict bool) (serviceId, envId string, importErr *serviceImportIdError) {
	parts := strings.SplitN(raw, ":", 2)
	serviceId = parts[0]

	if serviceId == "" {
		return "", "", &serviceImportIdError{
			summary: "Unexpected Import Identifier",
			detail:  fmt.Sprintf("Expected import identifier with format `service_id` or `service_id:environment_id`. Got: %q", raw),
		}
	}

	if len(parts) == 2 {
		envId = parts[1]
		if envId == "" {
			return "", "", &serviceImportIdError{
				summary: "Unexpected Import Identifier",
				detail:  fmt.Sprintf("Import identifier `service_id:environment_id` has an empty environment_id. Got: %q", raw),
			}
		}
		return serviceId, envId, nil
	}

	if strict {
		return "", "", &serviceImportIdError{
			summary: "environment_id required under strict env-scoping",
			detail:  fmt.Sprintf("Import identifier %q omits the environment_id. Under strict env-scoping (provider default) every service is fork-scoped and must carry the environment id it belongs to; without it, the subsequent plan would see environment_id as a change requiring replace and destroy the imported service. Use `terraform import railway_service.<name> <service_id>:<environment_id>` instead, or opt into permissive mode via `strict_env_scoping = false` on the provider block.", raw),
		}
	}

	return serviceId, "", nil
}

// ImportState accepts either `<service_id>` for a project-wide service
// (permissive env-scoping) or `<service_id>:<environment_id>` for a
// fork-scoped service. Under strict env-scoping (provider default) the
// environment_id part is required — importing a strict-mode service with just
// its id would leave environment_id null in state, and the subsequent plan
// would see the env_id in HCL as a change requiring replace, silently
// destroying and re-creating the service. Setting environment_id up front
// avoids that trap.
func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serviceId, envId, importErr := parseServiceImportId(req.ID, r.strictEnvScoping)
	if importErr != nil {
		resp.Diagnostics.AddError(importErr.summary, importErr.detail)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), serviceId)...)
	if envId != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), envId)...)
	}
}

// resolveTargetEnvironment returns the environment id the service is scoped to.
// Priority: the service's own environment_id (if set), then the project's
// default environment. Used by the inline-volume path (createVolume needs an
// environmentId) and by the volume readback in Read/Update.
func resolveTargetEnvironment(ctx context.Context, client graphql.Client, data *ServiceResourceModel) (string, error) {
	if !data.EnvironmentId.IsNull() && !data.EnvironmentId.IsUnknown() {
		return data.EnvironmentId.ValueString(), nil
	}

	_, defaultEnv, err := defaultEnvironmentForProject(ctx, client, data.ProjectId.ValueString())
	if err != nil {
		return "", err
	}
	return defaultEnv.Id, nil
}

// renameServiceVolumeIfNeeded renames the inline volume attached to a
// railway_service when — and only when — Railway's auto-assigned name does
// not already match the desired name.
//
// The read is retried across Railway's eventual-consistency window because
// getVolumeInstances does not immediately reflect a just-created volume;
// without the retry the lookup returns "not found" and the caller cannot
// tell whether the auto-name already matches, which is the exact case that
// causes updateVolume to fail with "A volume named X already exists in
// this project".
//
// Returns nil when the rename is unnecessary or succeeds; a non-nil error
// means either the current name could not be determined within the retry
// budget, or updateVolume genuinely failed.
func renameServiceVolumeIfNeeded(ctx context.Context, client graphql.Client, projectId, volId, desiredName string) error {
	var currentName string
	err := retryReadAfterCreateContext(ctx, 30*time.Second, func() error {
		response, readErr := getVolumeInstances(ctx, client, projectId)
		if readErr != nil {
			return readErr
		}
		for _, volume := range response.Project.Volumes.Edges {
			if volume.Node.Id == volId {
				currentName = volume.Node.Name
				return nil
			}
		}
		return &NotFoundError{ResourceType: "volume", Id: volId}
	})
	if err != nil {
		return fmt.Errorf("determining current volume name: %w", err)
	}

	if currentName == desiredName {
		return nil
	}

	_, err = updateVolume(ctx, client, volId, VolumeUpdateInput{Name: desiredName})
	return err
}

func getAndBuildVolumeInstance(ctx context.Context, client graphql.Client, projectId string, serviceId string, envId string, data *ServiceResourceModel) error {
	data.Volume = types.ObjectNull(volumeAttrTypes)

	response, err := getVolumeInstances(ctx, client, projectId)

	if err != nil {
		return err
	}

	for _, volume := range response.Project.Volumes.Edges {
		for _, volumeInstance := range volume.Node.VolumeInstances.Edges {
			if volumeInstance.Node.State == VolumeStateDeleted || volumeInstance.Node.State == VolumeStateDeleting {
				continue
			}
			if volumeInstance.Node.ServiceId == serviceId && volumeInstance.Node.EnvironmentId == envId {
				data.Volume = types.ObjectValueMust(
					volumeAttrTypes,
					map[string]attr.Value{
						"id":         types.StringValue(volume.Node.Id),
						"name":       types.StringValue(volume.Node.Name),
						"mount_path": types.StringValue(volumeInstance.Node.MountPath),
						"size":       types.Float64Value(float64(volumeInstance.Node.SizeMB)),
					},
				)
			}
		}
	}

	return nil
}
