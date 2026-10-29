package provider

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	resend "github.com/resend/resend-go/v4"
)

func TestFlattenRecords_empty(t *testing.T) {
	result, diags := flattenRecords(context.Background(), []resend.Record{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(result.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result.Elements()))
	}
}

func TestFlattenRecords_single(t *testing.T) {
	records := []resend.Record{
		{
			Record:   "SPF",
			Name:     "send",
			Type:     "TXT",
			Ttl:      "Auto",
			Status:   "verified",
			Value:    "v=spf1 include:amazonses.com ~all",
			Priority: json.Number("10"),
		},
	}

	result, diags := flattenRecords(context.Background(), records)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(result.Elements()) != 1 {
		t.Errorf("expected 1 element, got %d", len(result.Elements()))
	}
}

func TestFlattenRecords_noPriority(t *testing.T) {
	records := []resend.Record{
		{
			Record: "DKIM",
			Name:   "resend._domainkey",
			Type:   "CNAME",
			Ttl:    "Auto",
			Status: "pending",
			Value:  "some-dkim-value",
		},
	}

	result, diags := flattenRecords(context.Background(), records)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(result.Elements()) != 1 {
		t.Errorf("expected 1 element, got %d", len(result.Elements()))
	}
}

func TestFlattenRecords_multiple(t *testing.T) {
	records := []resend.Record{
		{
			Record:   "MX",
			Name:     "send",
			Type:     "MX",
			Ttl:      "Auto",
			Status:   "verified",
			Value:    "feedback-smtp.us-east-1.amazonses.com",
			Priority: json.Number("10"),
		},
		{
			Record: "SPF",
			Name:   "send",
			Type:   "TXT",
			Ttl:    "Auto",
			Status: "verified",
			Value:  "v=spf1 include:amazonses.com ~all",
		},
		{
			Record: "DKIM",
			Name:   "resend._domainkey",
			Type:   "CNAME",
			Ttl:    "Auto",
			Status: "verified",
			Value:  "some-dkim-value",
		},
	}

	result, diags := flattenRecords(context.Background(), records)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(result.Elements()) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result.Elements()))
	}
}

// --- Unit tests for CRUD error branches ---

// domainPlanVals returns tftypes values for a domain resource plan (Create).
func domainPlanVals(objType tftypes.Object) map[string]tftypes.Value {
	recordsType := objType.AttributeTypes["records"]
	return map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":               tftypes.NewValue(tftypes.String, "test.com"),
		"region":             tftypes.NewValue(tftypes.String, "us-east-1"),
		"custom_return_path": tftypes.NewValue(tftypes.String, nil),
		"open_tracking":      tftypes.NewValue(tftypes.Bool, false),
		"click_tracking":     tftypes.NewValue(tftypes.Bool, false),
		"tls":                tftypes.NewValue(tftypes.String, "opportunistic"),
		"status":             tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"created_at":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"records":            tftypes.NewValue(recordsType, tftypes.UnknownValue),
	}
}

// domainStateVals returns tftypes values for a domain resource state (Read/Update/Delete).
func domainStateVals(objType tftypes.Object) map[string]tftypes.Value {
	recordsType := objType.AttributeTypes["records"]
	return map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-id"),
		"name":               tftypes.NewValue(tftypes.String, "test.com"),
		"region":             tftypes.NewValue(tftypes.String, "us-east-1"),
		"custom_return_path": tftypes.NewValue(tftypes.String, ""),
		"open_tracking":      tftypes.NewValue(tftypes.Bool, false),
		"click_tracking":     tftypes.NewValue(tftypes.Bool, false),
		"tls":                tftypes.NewValue(tftypes.String, "opportunistic"),
		"status":             tftypes.NewValue(tftypes.String, "pending"),
		"created_at":         tftypes.NewValue(tftypes.String, "2024-01-01"),
		"records":            tftypes.NewValue(recordsType, []tftypes.Value{}),
	}
}

func TestDomainResource_Create_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
			return resend.CreateDomainResponse{}, errors.New("api error")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, domainPlanVals(objType)),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainResource_Create_updateError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
			return resend.CreateDomainResponse{Id: "test-id"}, nil
		},
		UpdateWithContextFn: func(_ context.Context, _ string, _ *resend.UpdateDomainRequest) (resend.Domain, error) {
			return resend.Domain{}, errors.New("update error")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	// Use non-default tracking settings to trigger the update path
	vals := domainPlanVals(objType)
	vals["open_tracking"] = tftypes.NewValue(tftypes.Bool, true)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, vals),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainResource_Create_getError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
			return resend.CreateDomainResponse{Id: "test-id"}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("get error")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, domainPlanVals(objType)),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainResource_Read_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("api error")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, domainStateVals(objType))

	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainResource_Read_notFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("not found")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, domainStateVals(objType))

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

func TestDomainResource_Update_updateError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		UpdateWithContextFn: func(_ context.Context, _ string, _ *resend.UpdateDomainRequest) (resend.Domain, error) {
			return resend.Domain{}, errors.New("update error")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	stateVals := domainStateVals(objType)

	req := resource.UpdateRequest{
		Plan:  testResourcePlan(schemaResp, objType, stateVals),
		State: testResourceState(schemaResp, objType, stateVals),
	}
	resp := resource.UpdateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Update(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainResource_Update_getError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		UpdateWithContextFn: func(_ context.Context, _ string, _ *resend.UpdateDomainRequest) (resend.Domain, error) {
			return resend.Domain{}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("get error")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	stateVals := domainStateVals(objType)

	req := resource.UpdateRequest{
		Plan:  testResourcePlan(schemaResp, objType, stateVals),
		State: testResourceState(schemaResp, objType, stateVals),
	}
	resp := resource.UpdateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Update(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainResource_Delete_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		RemoveWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("api error")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, domainStateVals(objType))

	req := resource.DeleteRequest{State: state}
	resp := resource.DeleteResponse{}

	r.Delete(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestDomainResource_Delete_notFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		RemoveWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("not found")
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, domainStateVals(objType))

	req := resource.DeleteRequest{State: state}
	resp := resource.DeleteResponse{}

	r.Delete(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for not-found delete, got: %v", resp.Diagnostics)
	}
}

// --- Constructor, Metadata, Configure, ImportState, populateFromDomain tests ---

func TestNewDomainResource(t *testing.T) {
	r := NewDomainResource()
	if r == nil {
		t.Error("expected non-nil resource")
	}
}

func TestDomainResource_Metadata(t *testing.T) {
	r := &domainResource{}
	req := resource.MetadataRequest{ProviderTypeName: "resend"}
	resp := resource.MetadataResponse{}
	r.Metadata(context.Background(), req, &resp)
	if resp.TypeName != "resend_domain" {
		t.Errorf("expected type name 'resend_domain', got %q", resp.TypeName)
	}
}

func TestDomainResource_Configure_nil(t *testing.T) {
	r := &domainResource{}
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

func TestDomainResource_Configure_valid(t *testing.T) {
	client := resend.NewClient("test-key")
	r := &domainResource{}
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

func TestDomainResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &domainResource{}
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

func TestDomainResourceModel_populateFromDomain(t *testing.T) {
	ctx := context.Background()
	domain := resend.Domain{
		Id:        "test-id",
		Name:      "test.com",
		Status:    "verified",
		Region:    "us-east-1",
		CreatedAt: "2024-01-01",
		Records:   []resend.Record{},
	}

	var model domainResourceModel
	diags := model.populateFromDomain(ctx, domain)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if model.ID.ValueString() != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "test.com" {
		t.Errorf("expected name 'test.com', got %q", model.Name.ValueString())
	}
}

// --- CRUD success path tests ---

func TestDomainResource_Create_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
			return resend.CreateDomainResponse{Id: "new-id"}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{
				Id:        "new-id",
				Name:      "test.com",
				Status:    "pending",
				Region:    "us-east-1",
				CreatedAt: "2024-01-01",
				Records:   []resend.Record{},
			}, nil
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	req := resource.CreateRequest{
		Plan: testResourcePlan(schemaResp, objType, domainPlanVals(objType)),
	}
	resp := resource.CreateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Create(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	var state domainResourceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &state)...)
	if state.ID.ValueString() != "new-id" {
		t.Errorf("expected ID 'new-id', got %q", state.ID.ValueString())
	}
}

func TestDomainResource_Create_withCustomReturnPath(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		CreateWithContextFn: func(_ context.Context, req *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
			if req.CustomReturnPath != "bounce.test.com" {
				t.Errorf("expected CustomReturnPath 'bounce.test.com', got %q", req.CustomReturnPath)
			}
			return resend.CreateDomainResponse{Id: "new-id"}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{
				Id:        "new-id",
				Name:      "test.com",
				Status:    "pending",
				Region:    "us-east-1",
				CreatedAt: "2024-01-01",
				Records:   []resend.Record{},
			}, nil
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	recordsType := objType.AttributeTypes["records"]
	vals := map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":               tftypes.NewValue(tftypes.String, "test.com"),
		"region":             tftypes.NewValue(tftypes.String, "us-east-1"),
		"custom_return_path": tftypes.NewValue(tftypes.String, "bounce.test.com"),
		"open_tracking":      tftypes.NewValue(tftypes.Bool, false),
		"click_tracking":     tftypes.NewValue(tftypes.Bool, false),
		"tls":                tftypes.NewValue(tftypes.String, "opportunistic"),
		"status":             tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"created_at":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"records":            tftypes.NewValue(recordsType, tftypes.UnknownValue),
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

func TestDomainResource_Create_withTracking(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
			return resend.CreateDomainResponse{Id: "new-id"}, nil
		},
		UpdateWithContextFn: func(_ context.Context, _ string, req *resend.UpdateDomainRequest) (resend.Domain, error) {
			if !req.OpenTracking {
				t.Error("expected OpenTracking true")
			}
			return resend.Domain{}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{
				Id:        "new-id",
				Name:      "test.com",
				Status:    "pending",
				Region:    "us-east-1",
				CreatedAt: "2024-01-01",
				Records:   []resend.Record{},
			}, nil
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)

	recordsType := objType.AttributeTypes["records"]
	vals := map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":               tftypes.NewValue(tftypes.String, "test.com"),
		"region":             tftypes.NewValue(tftypes.String, "us-east-1"),
		"custom_return_path": tftypes.NewValue(tftypes.String, nil),
		"open_tracking":      tftypes.NewValue(tftypes.Bool, true),
		"click_tracking":     tftypes.NewValue(tftypes.Bool, false),
		"tls":                tftypes.NewValue(tftypes.String, "opportunistic"),
		"status":             tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"created_at":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"records":            tftypes.NewValue(recordsType, tftypes.UnknownValue),
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

func TestDomainResource_Read_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{
				Id:        "test-id",
				Name:      "test.com",
				Status:    "verified",
				Region:    "us-east-1",
				CreatedAt: "2024-01-01",
				Records:   []resend.Record{},
			}, nil
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, domainStateVals(objType))

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

func TestDomainResource_Update_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		UpdateWithContextFn: func(_ context.Context, _ string, _ *resend.UpdateDomainRequest) (resend.Domain, error) {
			return resend.Domain{}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{
				Id:        "test-id",
				Name:      "test.com",
				Status:    "verified",
				Region:    "us-east-1",
				CreatedAt: "2024-01-01",
				Records:   []resend.Record{},
			}, nil
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	stateVals := domainStateVals(objType)

	req := resource.UpdateRequest{
		Plan:  testResourcePlan(schemaResp, objType, stateVals),
		State: testResourceState(schemaResp, objType, stateVals),
	}
	resp := resource.UpdateResponse{
		State: emptyResourceState(schemaResp),
	}

	r.Update(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestDomainResource_Delete_success(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		RemoveWithContextFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	r := &domainResource{domains: mock}
	schemaResp, objType := testResourceSchemaAndObjType(ctx, r)
	state := testResourceState(schemaResp, objType, domainStateVals(objType))

	req := resource.DeleteRequest{State: state}
	resp := resource.DeleteResponse{}

	r.Delete(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}
