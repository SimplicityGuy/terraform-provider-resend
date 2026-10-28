package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	resend "github.com/resend/resend-go/v3"
)

var _ resource.Resource = &domainResource{}
var _ resource.ResourceWithImportState = &domainResource{}

// NewDomainResource returns a new domain resource.
func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	domains resend.DomainsSvc
}

type domainResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Region           types.String `tfsdk:"region"`
	CustomReturnPath types.String `tfsdk:"custom_return_path"`
	OpenTracking     types.Bool   `tfsdk:"open_tracking"`
	ClickTracking    types.Bool   `tfsdk:"click_tracking"`
	Tls              types.String `tfsdk:"tls"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	Records          types.List   `tfsdk:"records"`
}

// populateFromDomain sets the common fields from a Resend API domain response.
func (m *domainResourceModel) populateFromDomain(ctx context.Context, domain resend.Domain) diag.Diagnostics {
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

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Resend sending domain. Creating the resource adds the domain to Resend, which generates the DNS records (`records`) that prove you own it. Publish those records with your DNS provider, then use `resend_domain_verification` to ask Resend to check them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Resend domain ID, a UUID. Use it with `terraform import` and wherever another resource needs the domain (`resend_domain_verification`, `resend_api_key`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain to send from, such as `updates.example.com`. Resend recommends a subdomain rather than the root domain. Changing it replaces the domain.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region Resend sends the domain's email from. Valid values: `us-east-1`, `eu-west-1`, `sa-east-1`, `ap-northeast-1`. Defaults to `us-east-1`. Changing it replaces the domain.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("us-east-1"),
				Validators: []validator.String{
					stringvalidator.OneOf("us-east-1", "eu-west-1", "sa-east-1", "ap-northeast-1"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"custom_return_path": schema.StringAttribute{
				MarkdownDescription: "The subdomain Resend uses for the Return-Path (bounce) address, such as `outbound`. Resend uses `send` when it is not set. It can only be set when the domain is created, so changing it replaces the domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"open_tracking": schema.BoolAttribute{
				MarkdownDescription: "Whether Resend tracks email opens for this domain. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"click_tracking": schema.BoolAttribute{
				MarkdownDescription: "Whether Resend tracks link clicks for this domain. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"tls": schema.StringAttribute{
				MarkdownDescription: "How Resend uses TLS when delivering mail. `opportunistic` tries a secure connection and falls back to sending unencrypted when the receiving server does not support TLS; `enforced` refuses to deliver without TLS. Valid values: `enforced`, `opportunistic`. Defaults to `opportunistic`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("opportunistic"),
				Validators: []validator.String{
					stringvalidator.OneOf("enforced", "opportunistic"),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The verification status of the domain: one of `not_started`, `pending`, `verified`, `partially_verified`, `partially_failed`, `failed` or `temporary_failure`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the domain was created, as reported by Resend.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "The DNS records to publish so Resend can verify the domain. Resend generates the set when the domain is created, so it is only known after the first apply.",
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

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.domains = configureDomainsSvc(req.ProviderData, &resp.Diagnostics)
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create domain
	createReq := &resend.CreateDomainRequest{
		Name:   plan.Name.ValueString(),
		Region: plan.Region.ValueString(),
	}
	if !plan.CustomReturnPath.IsNull() && !plan.CustomReturnPath.IsUnknown() {
		createReq.CustomReturnPath = plan.CustomReturnPath.ValueString()
	}

	createResp, err := retryOnRateLimit(ctx, func() (resend.CreateDomainResponse, error) {
		return r.domains.CreateWithContext(ctx, createReq)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating domain",
			"Could not create domain "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	domainID := createResp.Id

	// Update tracking/TLS settings only when non-default values are specified.
	// The Create API does not support these fields.
	needsUpdate := plan.OpenTracking.ValueBool() ||
		plan.ClickTracking.ValueBool() ||
		plan.Tls.ValueString() != "opportunistic"

	if needsUpdate {
		updateReq := &resend.UpdateDomainRequest{
			OpenTracking:  plan.OpenTracking.ValueBool(),
			ClickTracking: plan.ClickTracking.ValueBool(),
			Tls:           plan.Tls.ValueString(),
		}
		_, err = retryOnRateLimit(ctx, func() (resend.Domain, error) {
			return r.domains.UpdateWithContext(ctx, domainID, updateReq)
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating domain settings",
				"Domain "+domainID+" was created but failed to set tracking/TLS settings: "+err.Error(),
			)
			return
		}
	}

	// Read latest state
	domain, err := retryOnRateLimit(ctx, func() (resend.Domain, error) {
		return r.domains.GetWithContext(ctx, domainID)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading domain after creation",
			"Domain "+domainID+" was created but could not be read: "+err.Error(),
		)
		return
	}

	// Map response to state
	resp.Diagnostics.Append(plan.populateFromDomain(ctx, domain)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// custom_return_path: use plan value if set, otherwise leave computed
	if plan.CustomReturnPath.IsNull() || plan.CustomReturnPath.IsUnknown() {
		plan.CustomReturnPath = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := retryOnRateLimit(ctx, func() (resend.Domain, error) {
		return r.domains.GetWithContext(ctx, state.ID.ValueString())
	})
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading domain",
			"Could not read domain ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(state.populateFromDomain(ctx, domain)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Tracking/TLS fields are not returned by the SDK Domain struct.
	// Preserve existing state values (set during Create/Update).

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &resend.UpdateDomainRequest{
		OpenTracking:  plan.OpenTracking.ValueBool(),
		ClickTracking: plan.ClickTracking.ValueBool(),
		Tls:           plan.Tls.ValueString(),
	}

	_, err := retryOnRateLimit(ctx, func() (resend.Domain, error) {
		return r.domains.UpdateWithContext(ctx, state.ID.ValueString(), updateReq)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating domain",
			"Could not update domain ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Read latest state
	domain, err := retryOnRateLimit(ctx, func() (resend.Domain, error) {
		return r.domains.GetWithContext(ctx, state.ID.ValueString())
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading domain after update",
			"Domain "+state.ID.ValueString()+" was updated but could not be read: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.populateFromDomain(ctx, domain)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CustomReturnPath.IsNull() || plan.CustomReturnPath.IsUnknown() {
		plan.CustomReturnPath = state.CustomReturnPath
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := retryOnRateLimit(ctx, func() (bool, error) {
		return r.domains.RemoveWithContext(ctx, state.ID.ValueString())
	})
	if handleDeleteError(err, "domain", state.ID.ValueString(), &resp.Diagnostics) {
		return
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flattenRecords(_ context.Context, records []resend.Record) (types.List, diag.Diagnostics) {
	recordType := types.ObjectType{
		AttrTypes: domainRecordAttrTypes(),
	}

	if len(records) == 0 {
		return types.ListValueMust(recordType, []attr.Value{}), nil
	}

	var recordValues []attr.Value
	for _, record := range records {
		priority := ""
		if record.Priority != "" {
			priority = record.Priority.String()
		}

		recordObj, d := types.ObjectValue(domainRecordAttrTypes(), map[string]attr.Value{
			"record":   types.StringValue(record.Record),
			"name":     types.StringValue(record.Name),
			"type":     types.StringValue(record.Type),
			"ttl":      types.StringValue(record.Ttl),
			"status":   types.StringValue(record.Status),
			"value":    types.StringValue(record.Value),
			"priority": types.StringValue(priority),
		})
		if d.HasError() {
			return types.ListNull(recordType), d
		}
		recordValues = append(recordValues, recordObj)
	}

	return types.ListValue(recordType, recordValues)
}

func domainRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"record":   types.StringType,
		"name":     types.StringType,
		"type":     types.StringType,
		"ttl":      types.StringType,
		"status":   types.StringType,
		"value":    types.StringType,
		"priority": types.StringType,
	}
}
