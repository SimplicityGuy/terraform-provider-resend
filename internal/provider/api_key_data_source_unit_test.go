package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	resend "github.com/resend/resend-go/v4"
)

func apiKeyDataSourceConfigVals(id string) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, id),
		"name":       tftypes.NewValue(tftypes.String, nil),
		"created_at": tftypes.NewValue(tftypes.String, nil),
	}
}

func TestApiKeyDataSource_Read_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		ListWithContextFn: func(_ context.Context) (resend.ListApiKeysResponse, error) {
			return resend.ListApiKeysResponse{}, errors.New("api error")
		},
	}

	d := &apiKeyDataSource{apiKeys: mock}
	schemaResp, objType := testDataSourceSchemaAndObjType(ctx, d)

	req := datasource.ReadRequest{
		Config: testDataSourceConfig(schemaResp, objType, apiKeyDataSourceConfigVals("test-id")),
	}
	resp := datasource.ReadResponse{
		State: emptyDataSourceState(schemaResp),
	}

	d.Read(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestApiKeyDataSource_Read_notFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		ListWithContextFn: func(_ context.Context) (resend.ListApiKeysResponse, error) {
			// Return empty list (key not found)
			return resend.ListApiKeysResponse{Data: []resend.ApiKey{}}, nil
		},
	}

	d := &apiKeyDataSource{apiKeys: mock}
	schemaResp, objType := testDataSourceSchemaAndObjType(ctx, d)

	req := datasource.ReadRequest{
		Config: testDataSourceConfig(schemaResp, objType, apiKeyDataSourceConfigVals("test-id")),
	}
	resp := datasource.ReadResponse{
		State: emptyDataSourceState(schemaResp),
	}

	d.Read(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics for not-found API key")
	}
}

func TestNewApiKeyDataSource(t *testing.T) {
	d := NewApiKeyDataSource()
	if d == nil {
		t.Error("expected non-nil data source")
	}
}

func TestApiKeyDataSource_Metadata(t *testing.T) {
	d := &apiKeyDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "resend"}
	resp := datasource.MetadataResponse{}
	d.Metadata(context.Background(), req, &resp)
	if resp.TypeName != "resend_api_key" {
		t.Errorf("expected type name 'resend_api_key', got %q", resp.TypeName)
	}
}

func TestApiKeyDataSource_Configure_nil(t *testing.T) {
	d := &apiKeyDataSource{}
	req := datasource.ConfigureRequest{ProviderData: nil}
	resp := datasource.ConfigureResponse{}
	d.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if d.apiKeys != nil {
		t.Error("expected nil apiKeys for nil provider data")
	}
}

func TestApiKeyDataSource_Configure_valid(t *testing.T) {
	client := resend.NewClient("test-key")
	d := &apiKeyDataSource{}
	req := datasource.ConfigureRequest{ProviderData: client}
	resp := datasource.ConfigureResponse{}
	d.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if d.apiKeys == nil {
		t.Error("expected apiKeys to be set")
	}
}

func TestApiKeyDataSource_Read_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		ListWithContextFn: func(_ context.Context) (resend.ListApiKeysResponse, error) {
			return resend.ListApiKeysResponse{
				Data: []resend.ApiKey{
					{Id: "test-id", Name: "my-key", CreatedAt: "2024-01-01"},
				},
			}, nil
		},
	}

	d := &apiKeyDataSource{apiKeys: mock}
	schemaResp, objType := testDataSourceSchemaAndObjType(ctx, d)

	req := datasource.ReadRequest{
		Config: testDataSourceConfig(schemaResp, objType, apiKeyDataSourceConfigVals("test-id")),
	}
	resp := datasource.ReadResponse{
		State: emptyDataSourceState(schemaResp),
	}

	d.Read(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}
