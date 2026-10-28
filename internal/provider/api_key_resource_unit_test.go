package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	resend "github.com/resend/resend-go/v4"
)

func apiKeyPlanVals() map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":       tftypes.NewValue(tftypes.String, "test-key"),
		"permission": tftypes.NewValue(tftypes.String, "full_access"),
		"domain_id":  tftypes.NewValue(tftypes.String, nil),
		"token":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	}
}

func apiKeyStateVals() map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "test-id"),
		"name":       tftypes.NewValue(tftypes.String, "test-key"),
		"permission": tftypes.NewValue(tftypes.String, "full_access"),
		"domain_id":  tftypes.NewValue(tftypes.String, nil),
		"token":      tftypes.NewValue(tftypes.String, "re_xxx"),
	}
}

func TestApiKeyResource_Create_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateApiKeyRequest) (resend.CreateApiKeyResponse, error) {
			return resend.CreateApiKeyResponse{}, errors.New("api error")
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, apiKeyPlanVals()),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestApiKeyResource_Read_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		ListWithContextFn: func(_ context.Context) (resend.ListApiKeysResponse, error) {
			return resend.ListApiKeysResponse{}, errors.New("api error")
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, apiKeyStateVals())

	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestApiKeyResource_Read_notFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		ListWithContextFn: func(_ context.Context) (resend.ListApiKeysResponse, error) {
			// Return empty list (key not found)
			return resend.ListApiKeysResponse{Data: []resend.ApiKey{}}, nil
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, apiKeyStateVals())

	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for not-found, got: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed (null)")
	}
}

func TestApiKeyResource_Delete_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		RemoveWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("api error")
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, apiKeyStateVals())

	req := resource.DeleteRequest{State: state}
	resp := resource.DeleteResponse{}

	r.Delete(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestApiKeyResource_Delete_notFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		RemoveWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("not found")
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, apiKeyStateVals())

	req := resource.DeleteRequest{State: state}
	resp := resource.DeleteResponse{}

	r.Delete(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for not-found delete, got: %v", resp.Diagnostics)
	}
}

// --- Constructor, Metadata, Configure, no-op, ImportState tests ---

func TestNewApiKeyResource(t *testing.T) {
	r := NewApiKeyResource()
	if r == nil {
		t.Error("expected non-nil resource")
	}
}

func TestApiKeyResource_Metadata(t *testing.T) {
	r := &apiKeyResource{}
	req := resource.MetadataRequest{ProviderTypeName: "resend"}
	resp := resource.MetadataResponse{}
	r.Metadata(context.Background(), req, &resp)
	if resp.TypeName != "resend_api_key" {
		t.Errorf("expected type name 'resend_api_key', got %q", resp.TypeName)
	}
}

func TestApiKeyResource_Configure_nil(t *testing.T) {
	r := &apiKeyResource{}
	req := resource.ConfigureRequest{ProviderData: nil}
	resp := resource.ConfigureResponse{}
	r.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if r.apiKeys != nil {
		t.Error("expected nil apiKeys for nil provider data")
	}
}

func TestApiKeyResource_Configure_valid(t *testing.T) {
	client := resend.NewClient("test-key")
	r := &apiKeyResource{}
	req := resource.ConfigureRequest{ProviderData: client}
	resp := resource.ConfigureResponse{}
	r.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if r.apiKeys == nil {
		t.Error("expected apiKeys to be set")
	}
}

func TestApiKeyResource_Update_noop(t *testing.T) {
	r := &apiKeyResource{}
	req := resource.UpdateRequest{}
	resp := resource.UpdateResponse{}
	r.Update(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestApiKeyResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &apiKeyResource{}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.ImportStateRequest{ID: "test-id"}
	resp := resource.ImportStateResponse{
		State: testNullResourceState(schemaResp, objType),
	}

	r.ImportState(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

// --- CRUD success path tests ---

func TestApiKeyResource_Create_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateApiKeyRequest) (resend.CreateApiKeyResponse, error) {
			return resend.CreateApiKeyResponse{Id: "new-id", Token: "re_xxx"}, nil
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, apiKeyPlanVals()),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	var state apiKeyResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &state)...)
	if state.ID.ValueString() != "new-id" {
		t.Errorf("expected ID 'new-id', got %q", state.ID.ValueString())
	}
	if state.Token.ValueString() != "re_xxx" {
		t.Errorf("expected token 're_xxx', got %q", state.Token.ValueString())
	}
}

func TestApiKeyResource_Read_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		ListWithContextFn: func(_ context.Context) (resend.ListApiKeysResponse, error) {
			return resend.ListApiKeysResponse{
				Data: []resend.ApiKey{
					{Id: "test-id", Name: "my-key"},
				},
			}, nil
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, apiKeyStateVals())

	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	var result apiKeyResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &result)...)
	if result.Name.ValueString() != "my-key" {
		t.Errorf("expected name 'my-key', got %q", result.Name.ValueString())
	}
}

func TestApiKeyResource_Create_withDomainID(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		CreateWithContextFn: func(_ context.Context, req *resend.CreateApiKeyRequest) (resend.CreateApiKeyResponse, error) {
			if req.DomainId != "my-domain" {
				t.Errorf("expected DomainId 'my-domain', got %q", req.DomainId)
			}
			return resend.CreateApiKeyResponse{Id: "new-id", Token: "re_xxx"}, nil
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	vals := map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":       tftypes.NewValue(tftypes.String, "test-key"),
		"permission": tftypes.NewValue(tftypes.String, "sending_access"),
		"domain_id":  tftypes.NewValue(tftypes.String, "my-domain"),
		"token":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	}
	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, vals),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestApiKeyResource_Delete_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockApiKeysSvc{
		RemoveWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	r := &apiKeyResource{apiKeys: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, apiKeyStateVals())

	req := resource.DeleteRequest{State: state}
	resp := resource.DeleteResponse{}

	r.Delete(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}
