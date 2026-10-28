package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	resend "github.com/resend/resend-go/v3"
)

var _ datasource.DataSource = &apiKeyDataSource{}

// NewApiKeyDataSource returns a new API key data source.
func NewApiKeyDataSource() datasource.DataSource {
	return &apiKeyDataSource{}
}

type apiKeyDataSource struct {
	apiKeys resend.ApiKeysSvc
}

type apiKeyDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *apiKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *apiKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Resend API key by ID. Resend never returns a key's token after creation, so this data source exposes only its metadata.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Resend API key ID, a UUID.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the API key, shown in the Resend dashboard.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the API key was created, as reported by Resend.",
				Computed:            true,
			},
		},
	}
}

func (d *apiKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.apiKeys = configureApiKeysSvc(req.ProviderData, &resp.Diagnostics)
}

func (d *apiKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data apiKeyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No GET /api-keys/{id} endpoint; list all and find by ID.
	listResp, err := retryOnRateLimit(ctx, func() (resend.ListApiKeysResponse, error) {
		return d.apiKeys.ListWithContext(ctx)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing API keys",
			"Could not list API keys to find ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	for _, key := range listResp.Data {
		if key.Id == data.ID.ValueString() {
			data.Name = types.StringValue(key.Name)
			data.CreatedAt = types.StringValue(key.CreatedAt)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"API key not found",
		"Could not find API key with ID "+data.ID.ValueString(),
	)
}
