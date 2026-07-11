package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Khan/genqlient/graphql"
)

var (
	envVarName          = "RAILWAY_TOKEN"
	envVarStrictScoping = "RAILWAY_STRICT_ENV_SCOPING"
	errMissingAuthToken = "Required token could not be found. Please set the token using an input variable in the provider configuration block or by using the `" + envVarName + "` environment variable."
)

var uuidRegexp = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

func uuidRegex() *regexp.Regexp {
	return uuidRegexp
}

var _ provider.Provider = &RailwayProvider{}

type RailwayProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

var defaultAPIURL = "https://backboard.railway.app/graphql/v2?source=terraform_provider_railway"

type RailwayProviderModel struct {
	Token            types.String `tfsdk:"token"`
	APIURL           types.String `tfsdk:"api_url"`
	StrictEnvScoping types.Bool   `tfsdk:"strict_env_scoping"`
}

// RailwayProviderData is the value passed to every Resource / DataSource
// Configure() via req.ProviderData. It bundles the GraphQL client with
// provider-level flags that resources need at runtime.
type RailwayProviderData struct {
	Client           *graphql.Client
	StrictEnvScoping bool
}

func (p *RailwayProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "railway"
	resp.Version = p.version
}

func (p *RailwayProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway provider configuration.",
		Description:         "Railway provider configuration.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "The token used to authenticate with Railway.",
				Description:         "The token used to authenticate with Railway.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Override the Railway API URL. Used for testing.",
				Description:         "Override the Railway API URL. Used for testing.",
				Optional:            true,
			},
			"strict_env_scoping": schema.BoolAttribute{
				MarkdownDescription: "Enforce environment-scoped resource creation. When `true` (default), " +
					"`railway_service.environment_id` and `railway_environment.source_environment_id` are treated " +
					"as required, and any attempt to create a service or additional environment without them fails " +
					"at plan time. This prevents Railway's env-less mutation semantics (see " +
					"`ServiceCreateInput.environmentId` in the Railway schema) from silently creating services " +
					"across every non-fork environment. Set to `false` to opt out — you own the leak surface. " +
					"May also be set via the `" + envVarStrictScoping + "` environment variable.",
				Description: "Enforce environment-scoped resource creation. When true (default), " +
					"railway_service.environment_id and railway_environment.source_environment_id are treated as " +
					"required. Set to false to opt out. May also be set via the " + envVarStrictScoping + " env var.",
				Optional: true,
			},
		},
	}
}

func (p *RailwayProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RailwayProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	token := ""

	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	// If a token wasn't set in the provider configuration block, try and fetch it
	// from the environment variable.
	if token == "" {
		token = os.Getenv(envVarName)
	}

	// If we still don't have a token at this point, we return an error.
	if token == "" {
		resp.Diagnostics.AddError("Missing API token", errMissingAuthToken)
		return
	}

	httpClient := http.Client{
		Timeout: 30 * time.Second,
		Transport: &authedTransport{
			token:     token,
			userAgent: "terraform-provider-railway/" + p.version,
			wrapped:   http.DefaultTransport,
		},
	}

	apiURL := defaultAPIURL

	if !data.APIURL.IsNull() {
		apiURL = data.APIURL.ValueString()
	}

	// Resolve strict_env_scoping. Precedence: explicit HCL > env var > default (true).
	strictEnvScoping := true
	if !data.StrictEnvScoping.IsNull() {
		strictEnvScoping = data.StrictEnvScoping.ValueBool()
	} else if v := os.Getenv(envVarStrictScoping); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid "+envVarStrictScoping,
				fmt.Sprintf("Value %q is not a valid boolean. Use 'true' or 'false'.", v),
			)
			return
		}
		strictEnvScoping = parsed
	}

	client := graphql.NewClient(apiURL, &httpClient)

	providerData := &RailwayProviderData{
		Client:           &client,
		StrictEnvScoping: strictEnvScoping,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *RailwayProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewEnvironmentResource,
		NewServiceResource,
		NewServiceInstanceResource,
		NewVolumeResource,
		NewVolumeBackupScheduleResource,
		NewVariableResource,
		NewVariableCollectionResource,
		NewSharedVariableResource,
		NewCustomDomainResource,
		NewServiceDomainResource,
		NewTcpProxyResource,
		NewEgressGatewayResource,
		NewPrivateNetworkResource,
		NewPrivateNetworkEndpointResource,
		NewDeploymentTriggerResource,
		NewProjectTokenResource,
		NewTrustedDomainResource,
		NewNotificationRuleResource,
		NewBucketResource,
		NewSshPublicKeyResource,
		NewProjectMemberResource,
	}
}

func (p *RailwayProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
		NewEnvironmentDataSource,
		NewServiceDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RailwayProvider{
			version: version,
		}
	}
}

// providerDataFrom is the standard extractor every Resource / DataSource
// Configure() calls. It returns the RailwayProviderData bundle (client +
// flags) or reports a diagnostic and returns nil.
//
// Callers use the returned pointer's Client field for GraphQL and
// StrictEnvScoping to gate strict-mode diagnostics. Returning nil means
// "provider not yet configured" — callers should simply return without
// setting resource fields.
func providerDataFrom(providerData any, diags interface{ AddError(string, string) }) *RailwayProviderData {
	if providerData == nil {
		return nil
	}

	data, ok := providerData.(*RailwayProviderData)
	if !ok {
		diags.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *RailwayProviderData, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return data
}
