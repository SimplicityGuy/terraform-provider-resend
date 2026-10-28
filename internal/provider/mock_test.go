package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	resend "github.com/resend/resend-go/v4"
)

// The mocks must keep satisfying the SDK interfaces the provider depends on.
var (
	_ resend.DomainsSvc = (*mockDomainsSvc)(nil)
	_ resend.ApiKeysSvc = (*mockApiKeysSvc)(nil)
)

// mockDomainsSvc is a mock implementation of resend.DomainsSvc for testing.
type mockDomainsSvc struct {
	CreateWithContextFn func(ctx context.Context, params *resend.CreateDomainRequest) (resend.CreateDomainResponse, error)
	GetWithContextFn    func(ctx context.Context, domainId string) (resend.Domain, error)
	UpdateWithContextFn func(ctx context.Context, domainId string, params *resend.UpdateDomainRequest) (resend.Domain, error)
	RemoveWithContextFn func(ctx context.Context, domainId string) (bool, error)
	VerifyWithContextFn func(ctx context.Context, domainId string) (bool, error)
	ListWithContextFn   func(ctx context.Context) (resend.ListDomainsResponse, error)
}

func (m *mockDomainsSvc) CreateWithContext(ctx context.Context, params *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
	return m.CreateWithContextFn(ctx, params)
}

func (m *mockDomainsSvc) Create(params *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
	return m.CreateWithContextFn(context.Background(), params)
}

func (m *mockDomainsSvc) GetWithContext(ctx context.Context, domainId string) (resend.Domain, error) {
	return m.GetWithContextFn(ctx, domainId)
}

func (m *mockDomainsSvc) Get(domainId string) (resend.Domain, error) {
	return m.GetWithContextFn(context.Background(), domainId)
}

func (m *mockDomainsSvc) UpdateWithContext(ctx context.Context, domainId string, params *resend.UpdateDomainRequest) (resend.Domain, error) {
	return m.UpdateWithContextFn(ctx, domainId, params)
}

func (m *mockDomainsSvc) Update(domainId string, params *resend.UpdateDomainRequest) (resend.Domain, error) {
	return m.UpdateWithContextFn(context.Background(), domainId, params)
}

func (m *mockDomainsSvc) RemoveWithContext(ctx context.Context, domainId string) (bool, error) {
	return m.RemoveWithContextFn(ctx, domainId)
}

func (m *mockDomainsSvc) Remove(domainId string) (bool, error) {
	return m.RemoveWithContextFn(context.Background(), domainId)
}

func (m *mockDomainsSvc) VerifyWithContext(ctx context.Context, domainId string) (bool, error) {
	return m.VerifyWithContextFn(ctx, domainId)
}

func (m *mockDomainsSvc) Verify(domainId string) (bool, error) {
	return m.VerifyWithContextFn(context.Background(), domainId)
}

func (m *mockDomainsSvc) ListWithOptions(ctx context.Context, _ *resend.ListOptions) (resend.ListDomainsResponse, error) {
	return m.ListWithContextFn(ctx)
}

func (m *mockDomainsSvc) ListWithContext(ctx context.Context) (resend.ListDomainsResponse, error) {
	return m.ListWithContextFn(ctx)
}

func (m *mockDomainsSvc) List() (resend.ListDomainsResponse, error) {
	return m.ListWithContextFn(context.Background())
}

// mockApiKeysSvc is a mock implementation of resend.ApiKeysSvc for testing.
type mockApiKeysSvc struct {
	CreateWithContextFn func(ctx context.Context, params *resend.CreateApiKeyRequest) (resend.CreateApiKeyResponse, error)
	ListWithContextFn   func(ctx context.Context) (resend.ListApiKeysResponse, error)
	RemoveWithContextFn func(ctx context.Context, apiKeyId string) (bool, error)
	UpdateWithContextFn func(ctx context.Context, apiKeyId string, params *resend.UpdateApiKeyRequest) (resend.UpdateApiKeyResponse, error)
}

func (m *mockApiKeysSvc) CreateWithContext(ctx context.Context, params *resend.CreateApiKeyRequest) (resend.CreateApiKeyResponse, error) {
	return m.CreateWithContextFn(ctx, params)
}

func (m *mockApiKeysSvc) Create(params *resend.CreateApiKeyRequest) (resend.CreateApiKeyResponse, error) {
	return m.CreateWithContextFn(context.Background(), params)
}

func (m *mockApiKeysSvc) ListWithOptions(ctx context.Context, _ *resend.ListOptions) (resend.ListApiKeysResponse, error) {
	return m.ListWithContextFn(ctx)
}

func (m *mockApiKeysSvc) ListWithContext(ctx context.Context) (resend.ListApiKeysResponse, error) {
	return m.ListWithContextFn(ctx)
}

func (m *mockApiKeysSvc) List() (resend.ListApiKeysResponse, error) {
	return m.ListWithContextFn(context.Background())
}

func (m *mockApiKeysSvc) RemoveWithContext(ctx context.Context, apiKeyId string) (bool, error) {
	return m.RemoveWithContextFn(ctx, apiKeyId)
}

func (m *mockApiKeysSvc) Remove(apiKeyId string) (bool, error) {
	return m.RemoveWithContextFn(context.Background(), apiKeyId)
}

func (m *mockApiKeysSvc) UpdateWithContext(ctx context.Context, apiKeyId string, params *resend.UpdateApiKeyRequest) (resend.UpdateApiKeyResponse, error) {
	return m.UpdateWithContextFn(ctx, apiKeyId, params)
}

func (m *mockApiKeysSvc) Update(apiKeyId string, params *resend.UpdateApiKeyRequest) (resend.UpdateApiKeyResponse, error) {
	return m.UpdateWithContextFn(context.Background(), apiKeyId, params)
}

// --- Test helpers for constructing tfsdk types ---

// testResourceSchemaAndObjType returns the schema and tftypes.Object for a resource.
func testResourceSchemaAndObjType(ctx context.Context, r resource.Resource) (resource.SchemaResponse, tftypes.Object) {
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	attrTypes := map[string]tftypes.Type{}
	for name, attr := range resp.Schema.Attributes {
		attrTypes[name] = attr.GetType().TerraformType(ctx)
	}

	return resp, tftypes.Object{AttributeTypes: attrTypes}
}

// testDataSourceSchemaAndObjType returns the schema and tftypes.Object for a data source.
func testDataSourceSchemaAndObjType(ctx context.Context, d datasource.DataSource) (datasource.SchemaResponse, tftypes.Object) {
	var resp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &resp)

	attrTypes := map[string]tftypes.Type{}
	for name, attr := range resp.Schema.Attributes {
		attrTypes[name] = attr.GetType().TerraformType(ctx)
	}

	return resp, tftypes.Object{AttributeTypes: attrTypes}
}

// testResourceState creates a tfsdk.State for a resource.
func testResourceState(schemaResp resource.SchemaResponse, objType tftypes.Object, vals map[string]tftypes.Value) tfsdk.State {
	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

// testResourcePlan creates a tfsdk.Plan for a resource.
func testResourcePlan(schemaResp resource.SchemaResponse, objType tftypes.Object, vals map[string]tftypes.Value) tfsdk.Plan {
	return tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

// testDataSourceConfig creates a tfsdk.Config for a data source.
func testDataSourceConfig(schemaResp datasource.SchemaResponse, objType tftypes.Object, vals map[string]tftypes.Value) tfsdk.Config {
	return tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

// emptyResourceState creates an empty tfsdk.State for a resource response.
func emptyResourceState(schemaResp resource.SchemaResponse) tfsdk.State {
	return tfsdk.State{
		Schema: schemaResp.Schema,
	}
}

// emptyDataSourceState creates an empty tfsdk.State for a data source response.
func emptyDataSourceState(schemaResp datasource.SchemaResponse) tfsdk.State {
	return tfsdk.State{
		Schema: schemaResp.Schema,
	}
}

// testNullResourceState creates a null-valued tfsdk.State for use in ImportState tests.
func testNullResourceState(schemaResp resource.SchemaResponse, objType tftypes.Object) tfsdk.State {
	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, nil),
	}
}
