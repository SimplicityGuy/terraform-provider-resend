package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	resend "github.com/resend/resend-go/v4"
)

var _ resource.Resource = &apiKeyResource{}
var _ resource.ResourceWithImportState = &apiKeyResource{}

// NewApiKeyResource returns a new API key resource.
func NewApiKeyResource() resource.Resource {
	return &apiKeyResource{}
}

type apiKeyResource struct {
	apiKeys resend.ApiKeysSvc
}

type apiKeyResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Permission types.String `tfsdk:"permission"`
	DomainID   types.String `tfsdk:"domain_id"`
	Token      types.String `tfsdk:"token"`
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Resend API key. Resend shows a key's token only once, when the key is created, and the provider keeps it in state as the sensitive `token` attribute. Changing any argument replaces the key, which issues a new token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Resend API key ID, a UUID. This is not the secret token.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the API key, shown in the Resend dashboard. Changing it replaces the key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission": schema.StringAttribute{
				MarkdownDescription: "What the key may do. `full_access` can create, read, update and delete any resource; `sending_access` can only send email. Valid values: `full_access`, `sending_access`. Defaults to `full_access`. Changing it replaces the key.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("full_access"),
				Validators: []validator.String{
					stringvalidator.OneOf("full_access", "sending_access"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain_id": schema.StringAttribute{
				MarkdownDescription: "Restricts the key to sending from one domain, by domain ID. Resend only applies it when `permission` is `sending_access`. Changing it replaces the key.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The secret API key (`re_...`). Resend returns it only when the key is created, so it is empty for an imported key and can't be recovered if lost from state.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.apiKeys = configureApiKeysSvc(req.ProviderData, &resp.Diagnostics)
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &resend.CreateApiKeyRequest{
		Name:       plan.Name.ValueString(),
		Permission: plan.Permission.ValueString(),
	}
	if !plan.DomainID.IsNull() {
		createReq.DomainId = plan.DomainID.ValueString()
	}

	createResp, err := retryOnRateLimit(ctx, func() (resend.CreateApiKeyResponse, error) {
		return r.apiKeys.CreateWithContext(ctx, createReq)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating API key",
			"Could not create API key "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(createResp.Id)
	plan.Token = types.StringValue(createResp.Token)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No GET /api-keys/{id} endpoint exists; list all and find by ID.
	listResp, err := retryOnRateLimit(ctx, func() (resend.ListApiKeysResponse, error) {
		return r.apiKeys.ListWithContext(ctx)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing API keys",
			"Could not list API keys to find ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	var found bool
	for _, key := range listResp.Data {
		if key.Id == state.ID.ValueString() {
			state.Name = types.StringValue(key.Name)
			found = true
			break
		}
	}

	if !found {
		// Resource was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Token is only returned on create; preserve state value.
	// Permission and DomainID are not returned by List; preserve state values.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiKeyResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes are ForceNew; Update should never be called.
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := retryOnRateLimit(ctx, func() (bool, error) {
		return r.apiKeys.RemoveWithContext(ctx, state.ID.ValueString())
	})
	if handleDeleteError(err, "API key", state.ID.ValueString(), &resp.Diagnostics) {
		return
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
