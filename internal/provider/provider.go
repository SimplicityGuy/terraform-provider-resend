package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ResendProvider{}

// ResendProvider defines the provider implementation.
type ResendProvider struct {
	version string
}

// ResendProviderModel describes the provider data model.
type ResendProviderModel struct {
	ApiKey types.String `tfsdk:"api_key"`
}

func (p *ResendProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "resend"
	resp.Version = p.version
}

func (p *ResendProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Resend provider manages Resend sending domains and API keys with Terraform or OpenTofu.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The Resend API key the provider authenticates with. It needs `full_access` permission to manage domains and API keys. Falls back to the `RESEND_API_KEY` environment variable; one of the two is required.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *ResendProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ResendProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("RESEND_API_KEY")
	if !data.ApiKey.IsNull() {
		apiKey = data.ApiKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider requires a Resend API key. Set it in the provider configuration or use the RESEND_API_KEY environment variable.",
		)
		return
	}

	client := newResendClient(apiKey, p.version)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ResendProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
		NewApiKeyResource,
		NewDomainVerificationResource,
	}
}

func (p *ResendProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
		NewApiKeyDataSource,
	}
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ResendProvider{
			version: version,
		}
	}
}
