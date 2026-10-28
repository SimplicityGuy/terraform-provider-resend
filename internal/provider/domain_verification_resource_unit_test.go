package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	resend "github.com/resend/resend-go/v4"
)

func verificationPlanVals(domainID string) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"domain_id": tftypes.NewValue(tftypes.String, domainID),
		"status":    tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	}
}

func verificationStateVals(domainID, status string) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"domain_id": tftypes.NewValue(tftypes.String, domainID),
		"status":    tftypes.NewValue(tftypes.String, status),
	}
}

func TestDomainVerificationResource_Create_verifyError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		VerifyWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("verify error")
		},
	}

	r := &domainVerificationResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, verificationPlanVals("test-domain-id")),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainVerificationResource_Create_getError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		VerifyWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("get error")
		},
	}

	r := &domainVerificationResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, verificationPlanVals("test-domain-id")),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainVerificationResource_Read_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("api error")
		},
	}

	r := &domainVerificationResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, verificationStateVals("test-domain-id", "pending"))

	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainVerificationResource_Read_notFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("not found")
		},
	}

	r := &domainVerificationResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, verificationStateVals("test-domain-id", "pending"))

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

// --- Constructor, Metadata, Configure, no-op tests ---

func TestNewDomainVerificationResource(t *testing.T) {
	r := NewDomainVerificationResource()
	if r == nil {
		t.Error("expected non-nil resource")
	}
}

func TestDomainVerificationResource_Metadata(t *testing.T) {
	r := &domainVerificationResource{}
	req := resource.MetadataRequest{ProviderTypeName: "resend"}
	resp := resource.MetadataResponse{}
	r.Metadata(context.Background(), req, &resp)
	if resp.TypeName != "resend_domain_verification" {
		t.Errorf("expected type name 'resend_domain_verification', got %q", resp.TypeName)
	}
}

func TestDomainVerificationResource_Configure_nil(t *testing.T) {
	r := &domainVerificationResource{}
	req := resource.ConfigureRequest{ProviderData: nil}
	resp := resource.ConfigureResponse{}
	r.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if r.domains != nil {
		t.Error("expected nil domains for nil provider data")
	}
}

func TestDomainVerificationResource_Configure_valid(t *testing.T) {
	client := resend.NewClient("test-key")
	r := &domainVerificationResource{}
	req := resource.ConfigureRequest{ProviderData: client}
	resp := resource.ConfigureResponse{}
	r.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if r.domains == nil {
		t.Error("expected domains to be set")
	}
}

func TestDomainVerificationResource_Update_noop(t *testing.T) {
	r := &domainVerificationResource{}
	req := resource.UpdateRequest{}
	resp := resource.UpdateResponse{}
	r.Update(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestDomainVerificationResource_Delete_noop(t *testing.T) {
	r := &domainVerificationResource{}
	req := resource.DeleteRequest{}
	resp := resource.DeleteResponse{}
	r.Delete(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
}

// --- CRUD success path tests ---

func TestDomainVerificationResource_Create_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		VerifyWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{Status: "verified"}, nil
		},
	}

	r := &domainVerificationResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, verificationPlanVals("test-domain-id")),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestDomainVerificationResource_Read_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{Status: "verified"}, nil
		},
	}

	r := &domainVerificationResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, verificationStateVals("test-domain-id", "pending"))

	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	if resp.State.Raw.IsNull() {
		t.Error("expected state to not be null")
	}
}
