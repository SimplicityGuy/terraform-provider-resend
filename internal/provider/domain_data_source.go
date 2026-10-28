package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	resend "github.com/resend/resend-go/v4"
)

var _ datasource.DataSource = &domainDataSource{}

// NewDomainDataSource returns a new domain data source.
func NewDomainDataSource() datasource.DataSource {
	return &domainDataSource{}
}

type domainDataSource struct {
	domains resend.DomainsSvc
}

type domainDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Region    types.String `tfsdk:"region"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	Records   types.List   `tfsdk:"records"`
}

// populateFromDomain sets the common fields from a Resend API domain response.
func (m *domainDataSourceModel) populateFromDomain(ctx context.Context, domain resend.Domain) diag.Diagnostics {
	m.ID = types.StringValue(domain.Id)
	m.Name = types.StringValue(domain.Name)
	m.Status = types.StringValue(domain.Status)
	m.Region = types.StringValue(domain.Region)
	m.CreatedAt = types.StringValue(domain.CreatedAt)

	records, diags := flattenRecords(ctx, domain.Records)
	if !diags.HasError() {
		m.Records = records
	}
	return diags
}

func (d *domainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Resend domain by ID, including the DNS records Resend needs to verify it. Use it for a domain managed outside this configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Resend domain ID, a UUID.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain name, such as `updates.example.com`.",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region Resend sends the domain's email from: `us-east-1`, `eu-west-1`, `sa-east-1` or `ap-northeast-1`.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The verification status of the domain: one of `not_started`, `pending`, `verified`, `partially_verified`, `partially_failed`, `failed` or `temporary_failure`.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the domain was created, as reported by Resend.",
				Computed:            true,
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "The DNS records to publish so Resend can verify the domain.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"record": schema.StringAttribute{
							MarkdownDescription: "What the record is for, as Resend labels it: `SPF`, `DKIM`, and, depending on the domain's settings, records such as `Tracking` or `Receiving`.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The record name relative to the root domain, such as `send` or `resend._domainkey` (`send.updates` for the subdomain `updates.example.com`). Some records, such as `Tracking`, are returned fully qualified.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The DNS record type, such as `MX`, `TXT` or `CNAME`.",
							Computed:            true,
						},
						"ttl": schema.StringAttribute{
							MarkdownDescription: "The time to live Resend recommends. `Auto` means the DNS provider's default.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The verification status of this record: one of `not_started`, `pending`, `verified`, `failed` or `temporary_failure`.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The record content. Publish it exactly as returned: the SPF TXT value includes its surrounding quotes.",
							Computed:            true,
						},
						"priority": schema.StringAttribute{
							MarkdownDescription: "The MX priority, as a string (for example `10`). Empty for record types that have no priority.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *domainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.domains = configureDomainsSvc(req.ProviderData, &resp.Diagnostics)
}

func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := retryOnRateLimit(ctx, func() (resend.Domain, error) {
		return d.domains.GetWithContext(ctx, data.ID.ValueString())
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading domain",
			"Could not read domain ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(data.populateFromDomain(ctx, domain)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
