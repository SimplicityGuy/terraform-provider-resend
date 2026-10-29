package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	resend "github.com/resend/resend-go/v4"
)

var _ resource.Resource = &domainVerificationResource{}

// NewDomainVerificationResource returns a new domain verification resource.
func NewDomainVerificationResource() resource.Resource {
	return &domainVerificationResource{}
}

type domainVerificationResource struct {
	domains resend.DomainsSvc
}

type domainVerificationResourceModel struct {
	DomainID types.String `tfsdk:"domain_id"`
	Status   types.String `tfsdk:"status"`
}

func (r *domainVerificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_verification"
}

func (r *domainVerificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Asks Resend to verify a domain's DNS records. Creating the resource sends one verification request and records the domain's status; it does not wait for verification to finish, and Resend keeps checking in the background. Publish the domain's `records` first (make this resource depend on your DNS records), or the check starts before the records exist. Destroying the resource only removes it from state: a verification can't be undone.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the domain to verify, usually `resend_domain.<name>.id`. Changing it replaces the resource, which sends a new verification request.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The domain's verification status, refreshed on every read: one of `not_started`, `pending`, `verified`, `partially_verified`, `partially_failed`, `failed` or `temporary_failure`.",
				Computed:            true,
			},
		},
	}
}

func (r *domainVerificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.domains = configureDomainsSvc(req.ProviderData, &resp.Diagnostics)
}

func (r *domainVerificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainVerificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueString()

	// Trigger verification
	_, err := retryOnRateLimit(ctx, func() (bool, error) {
		return r.domains.VerifyWithContext(ctx, domainID)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error verifying domain",
			"Could not trigger verification for domain ID "+domainID+": "+err.Error(),
		)
		return
	}

	// Read domain to get current status
	domain, err := retryOnRateLimit(ctx, func() (resend.Domain, error) {
		return r.domains.GetWithContext(ctx, domainID)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading domain after verification",
			"Verification was triggered for domain "+domainID+" but could not read status: "+err.Error(),
		)
		return
	}

	plan.Status = types.StringValue(domain.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainVerificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainVerificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := retryOnRateLimit(ctx, func() (resend.Domain, error) {
		return r.domains.GetWithContext(ctx, state.DomainID.ValueString())
	})
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading domain",
			"Could not read domain ID "+state.DomainID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Status = types.StringValue(domain.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainVerificationResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// domain_id is ForceNew; Update should never be called.
}

func (r *domainVerificationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Verification cannot be undone; no-op.
}
