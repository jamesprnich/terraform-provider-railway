package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

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

var _ resource.Resource = &ServiceInstanceResource{}
var _ resource.ResourceWithImportState = &ServiceInstanceResource{}

func NewServiceInstanceResource() resource.Resource {
	return &ServiceInstanceResource{}
}

type ServiceInstanceResource struct {
	client *graphql.Client
}

type ServiceInstanceResourceModel struct {
	Id               types.String  `tfsdk:"id"`
	ServiceId        types.String  `tfsdk:"service_id"`
	EnvironmentId    types.String  `tfsdk:"environment_id"`
	SourceImage      types.String  `tfsdk:"source_image"`
	SourceRepo       types.String  `tfsdk:"source_repo"`
	RootDirectory    types.String  `tfsdk:"root_directory"`
	ConfigPath       types.String  `tfsdk:"config_path"`
	BuildCommand     types.String  `tfsdk:"build_command"`
	StartCommand     types.String  `tfsdk:"start_command"`
	Region           types.String  `tfsdk:"region"`
	CronSchedule     types.String  `tfsdk:"cron_schedule"`
	HealthcheckPath  types.String  `tfsdk:"healthcheck_path"`
	NumReplicas      types.Int64   `tfsdk:"num_replicas"`
	VCPUs            types.Float64 `tfsdk:"vcpus"`
	MemoryGB         types.Float64 `tfsdk:"memory_gb"`
	SleepApplication types.Bool    `tfsdk:"sleep_application"`
}

func (r *ServiceInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_instance"
}

func (r *ServiceInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service instance. Configures a service in a specific environment, including source, build, deploy settings, and resource limits.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service instance (service_id:environment_id).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service.",
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
			"source_image": schema.StringAttribute{
				MarkdownDescription: "Docker image to use as the source (e.g. `postgres:17`). Conflicts with `source_repo`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo")),
				},
			},
			"source_repo": schema.StringAttribute{
				MarkdownDescription: "GitHub repository to use as the source (e.g. `owner/repo`). Conflicts with `source_image`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(3),
				},
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Root directory for the service within the repository.",
				Optional:            true,
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "Path to the Railway config file (e.g. `backend/railway.toml`).",
				Optional:            true,
			},
			"build_command": schema.StringAttribute{
				MarkdownDescription: "Custom build command.",
				Optional:            true,
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "Custom start command.",
				Optional:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region to deploy the service instance in.",
				Optional:            true,
				Computed:            true,
			},
			"cron_schedule": schema.StringAttribute{
				MarkdownDescription: "Cron schedule for the service.",
				Optional:            true,
			},
			"healthcheck_path": schema.StringAttribute{
				MarkdownDescription: "HTTP path for health checks.",
				Optional:            true,
			},
			"num_replicas": schema.Int64Attribute{
				MarkdownDescription: "Number of replicas.",
				Optional:            true,
				Computed:            true,
			},
			"vcpus": schema.Float64Attribute{
				MarkdownDescription: "Number of vCPUs to allocate (e.g. `0.5`, `1`, `2`). Maps to Railway's container CPU limit.",
				Optional:            true,
			},
			"memory_gb": schema.Float64Attribute{
				MarkdownDescription: "Amount of memory in GB to allocate (e.g. `0.5`, `1`, `2`). Maps to Railway's container memory limit.",
				Optional:            true,
			},
			"sleep_application": schema.BoolAttribute{
				MarkdownDescription: "Whether the service should sleep when inactive.",
				Optional:            true,
			},
		},
	}
}

func (r *ServiceInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceInstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("%s:%s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString()))

	input := buildServiceInstanceUpdateInput(data)

	_, err := updateServiceInstanceInEnvironment(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to configure service instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "configured service instance")

	// Resolve Unknown computed values before saving state.
	// New service instances may not have these set yet.
	if data.Region.IsUnknown() {
		data.Region = types.StringNull()
	}
	if data.NumReplicas.IsUnknown() {
		data.NumReplicas = types.Int64Null()
	}

	// Save state immediately so Terraform tracks this resource.
	// If any subsequent step fails, the resource will be tainted
	// and scheduled for destroy+recreate on the next apply.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Set resource limits if specified.
	// Retry with backoff — the service instance may not be immediately
	// queryable in a newly-created environment.
	if !data.VCPUs.IsNull() || !data.MemoryGB.IsNull() {
		err := retryFindContext(ctx, 15*time.Second, func() error {
			return r.updateLimits(ctx, data)
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set resource limits, got error: %s", err))
			return
		}
	}

	// Trigger an initial deployment so the service actually starts running
	_, err = deployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to deploy service instance, got error: %s", err))
		return
	}

	// Read back the state
	err = r.readServiceInstanceState(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service instance after creation, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceInstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readServiceInstanceState(ctx, data)

	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service instance, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceInstanceResourceModel
	var state *ServiceInstanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = state.Id

	input := buildServiceInstanceUpdateInput(data)

	_, err := updateServiceInstanceInEnvironment(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated service instance")

	// Update resource limits if changed
	limitsChanged := !state.VCPUs.Equal(data.VCPUs) || !state.MemoryGB.Equal(data.MemoryGB)
	if limitsChanged && (!data.VCPUs.IsNull() || !data.MemoryGB.IsNull()) {
		err := r.updateLimits(ctx, data)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update resource limits, got error: %s", err))
			return
		}
	}

	// Redeploy the instance to pick up changes
	_, err = redeployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to redeploy service instance after update, got error: %s", err))
		return
	}

	// Read back the state
	err = r.readServiceInstanceState(ctx, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service instance after update, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Service instances can't truly be deleted — they exist implicitly.
	// On delete, we reset the configuration to defaults.
	var data *ServiceInstanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ServiceInstanceUpdateInput{}

	_, err := updateServiceInstanceInEnvironment(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), input)

	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset service instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "reset service instance to defaults")
}

func (r *ServiceInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

// updateLimits calls the dedicated serviceInstanceLimitsUpdate mutation.
func (r *ServiceInstanceResource) updateLimits(ctx context.Context, data *ServiceInstanceResourceModel) error {
	limitsInput := ServiceInstanceLimitsUpdateInput{
		EnvironmentId: data.EnvironmentId.ValueString(),
		ServiceId:     data.ServiceId.ValueString(),
	}

	if !data.VCPUs.IsNull() {
		vcpus := data.VCPUs.ValueFloat64()
		limitsInput.VCPUs = &vcpus
	}

	if !data.MemoryGB.IsNull() {
		memoryGB := data.MemoryGB.ValueFloat64()
		limitsInput.MemoryGB = &memoryGB
	}

	_, err := updateServiceInstanceLimits(ctx, *r.client, limitsInput)

	if err != nil {
		return err
	}

	tflog.Trace(ctx, "updated resource limits")
	return nil
}

// buildServiceInstanceUpdateInput converts the Terraform model to a ServiceInstanceUpdateInput.
func buildServiceInstanceUpdateInput(data *ServiceInstanceResourceModel) ServiceInstanceUpdateInput {
	var input ServiceInstanceUpdateInput

	if !data.RootDirectory.IsNull() {
		input.RootDirectory = data.RootDirectory.ValueStringPointer()
	}

	if !data.ConfigPath.IsNull() {
		input.RailwayConfigFile = data.ConfigPath.ValueStringPointer()
	}

	if !data.CronSchedule.IsNull() {
		input.CronSchedule = data.CronSchedule.ValueStringPointer()
	}

	if !data.BuildCommand.IsNull() {
		input.BuildCommand = data.BuildCommand.ValueStringPointer()
	}

	if !data.StartCommand.IsNull() {
		input.StartCommand = data.StartCommand.ValueStringPointer()
	}

	if !data.Region.IsNull() && !data.Region.IsUnknown() {
		input.Region = data.Region.ValueStringPointer()
	}

	if !data.NumReplicas.IsNull() && !data.NumReplicas.IsUnknown() {
		numReplicas := int(data.NumReplicas.ValueInt64())
		input.NumReplicas = &numReplicas
	}

	if !data.HealthcheckPath.IsNull() {
		input.HealthcheckPath = data.HealthcheckPath.ValueStringPointer()
	}

	if !data.SleepApplication.IsNull() {
		sleepApp := data.SleepApplication.ValueBool()
		input.SleepApplication = &sleepApp
	}

	// Handle source via environment-scoped Source input (not serviceConnect, which is service-level)
	if !data.SourceImage.IsNull() {
		input.Source = &ServiceSourceInput{
			Image: data.SourceImage.ValueStringPointer(),
		}
	} else if !data.SourceRepo.IsNull() {
		input.Source = &ServiceSourceInput{
			Repo: data.SourceRepo.ValueStringPointer(),
		}
	}

	return input
}

// readServiceInstanceState queries the service instance and populates the model.
func (r *ServiceInstanceResource) readServiceInstanceState(ctx context.Context, data *ServiceInstanceResourceModel) error {
	response, err := getServiceInstanceDetailed(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		return err
	}

	instance := response.ServiceInstance

	data.Id = types.StringValue(fmt.Sprintf("%s:%s", data.ServiceId.ValueString(), data.EnvironmentId.ValueString()))

	// Source — Optional-only fields: only update from API if user configured them.
	// The source may be set at the service level (via serviceConnect) but the
	// service_instance resource shouldn't adopt it into state unless explicitly configured.
	if instance.Source != nil {
		if instance.Source.Image != nil && len(*instance.Source.Image) > 0 && !data.SourceImage.IsNull() {
			data.SourceImage = types.StringValue(*instance.Source.Image)
		}

		if instance.Source.Repo != nil && len(*instance.Source.Repo) > 0 && !data.SourceRepo.IsNull() {
			data.SourceRepo = types.StringValue(*instance.Source.Repo)
		}
	}

	// For Optional-only fields (not Computed): only update from the API when
	// it returns a non-empty value. Never null out a user-configured value —
	// the API may not echo these fields back immediately after a redeploy.

	// Build config — only update from API if user configured them.
	// These fields may be set at the service level (via railway_service) and
	// the service_instance resource shouldn't adopt them unless explicitly configured.
	if instance.RootDirectory != nil && len(*instance.RootDirectory) > 0 && !data.RootDirectory.IsNull() {
		data.RootDirectory = types.StringValue(*instance.RootDirectory)
	}

	if instance.RailwayConfigFile != nil && len(*instance.RailwayConfigFile) > 0 && !data.ConfigPath.IsNull() {
		data.ConfigPath = types.StringValue(*instance.RailwayConfigFile)
	}

	if instance.BuildCommand != nil && len(*instance.BuildCommand) > 0 && !data.BuildCommand.IsNull() {
		data.BuildCommand = types.StringValue(*instance.BuildCommand)
	}

	if instance.StartCommand != nil && len(*instance.StartCommand) > 0 && !data.StartCommand.IsNull() {
		data.StartCommand = types.StringValue(*instance.StartCommand)
	}

	// Deploy config — region and num_replicas are Computed, so resolve Unknowns
	if instance.Region != nil && len(*instance.Region) > 0 {
		data.Region = types.StringValue(*instance.Region)
	} else if data.Region.IsUnknown() {
		data.Region = types.StringNull()
	}

	if instance.CronSchedule != nil && len(*instance.CronSchedule) > 0 && !data.CronSchedule.IsNull() {
		data.CronSchedule = types.StringValue(*instance.CronSchedule)
	}

	if instance.HealthcheckPath != nil && len(*instance.HealthcheckPath) > 0 && !data.HealthcheckPath.IsNull() {
		data.HealthcheckPath = types.StringValue(*instance.HealthcheckPath)
	}

	if instance.NumReplicas != nil {
		data.NumReplicas = types.Int64Value(int64(*instance.NumReplicas))
	} else if data.NumReplicas.IsUnknown() {
		data.NumReplicas = types.Int64Null()
	}

	if instance.SleepApplication != nil && !data.SleepApplication.IsNull() {
		data.SleepApplication = types.BoolValue(*instance.SleepApplication)
	}

	// vCPUs and memory_gb are write-only via serviceInstanceLimitsUpdate.
	// The ServiceInstance type does not expose these fields in reads,
	// so we preserve the values from Terraform state/plan.

	return nil
}
