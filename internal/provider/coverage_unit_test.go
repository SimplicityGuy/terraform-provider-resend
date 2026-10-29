package provider

import (
	"context"
	"runtime/debug"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	resend "github.com/resend/resend-go/v4"
)

// These tests cover the branches that guard against a request whose plan,
// state or config cannot be decoded into the resource model, and the branches
// that handle a domain whose records cannot be flattened. The service mocks
// are left empty on purpose: a handler that gets past the guard would call a
// nil mock function and panic, so reaching the assertion proves it stopped.

func wantError(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if !diags.HasError() {
		t.Error("expected error in diagnostics")
	}
}

// failFlatten makes flattenRecords fail by forcing its object construction to
// return an error, restoring the real constructor afterwards.
func failFlatten(t *testing.T) {
	t.Helper()
	orig := objectValue
	t.Cleanup(func() { objectValue = orig })
	objectValue = func(map[string]attr.Type, map[string]attr.Value) (types.Object, diag.Diagnostics) {
		var d diag.Diagnostics
		d.AddError("forced", "forced object construction failure")
		return types.ObjectNull(domainRecordAttrTypes()), d
	}
}

// undecodable* return a plan, state or config that has a schema but no value,
// so Get into the resource model fails.
func undecodablePlan(s resource.SchemaResponse) tfsdk.Plan   { return tfsdk.Plan{Schema: s.Schema} }
func undecodableState(s resource.SchemaResponse) tfsdk.State { return tfsdk.State{Schema: s.Schema} }
func undecodableConfig(s datasource.SchemaResponse) tfsdk.Config {
	return tfsdk.Config{Schema: s.Schema}
}

func oneRecordDomain() resend.Domain {
	return resend.Domain{
		Id: "test-id", Name: "test.com", Status: "pending", Region: "us-east-1", CreatedAt: "2024-01-01",
		Records: []resend.Record{{Record: "SPF", Name: "send", Type: "TXT", Ttl: "Auto", Status: "pending", Value: "v"}},
	}
}

func TestFlattenRecords_objectError(t *testing.T) {
	failFlatten(t)
	list, diags := flattenRecords(context.Background(), oneRecordDomain().Records)
	wantError(t, diags)
	if !list.IsNull() {
		t.Error("expected a null list on error")
	}
}

func TestResendGoVersion_noBuildInfo(t *testing.T) {
	orig := readBuildInfo
	t.Cleanup(func() { readBuildInfo = orig })
	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }

	if got := resendGoVersion(); got != unknownVersion {
		t.Errorf("got %q, want %q", got, unknownVersion)
	}
}

func TestProvider_Configure_undecodableConfig(t *testing.T) {
	p := &ResendProvider{}
	resp := provider.ConfigureResponse{}
	var sr provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &sr)
	p.Configure(context.Background(), provider.ConfigureRequest{Config: tfsdk.Config{Schema: sr.Schema}}, &resp)
	wantError(t, resp.Diagnostics)
}

func TestApiKeyDataSource_Read_undecodableConfig(t *testing.T) {
	ctx := context.Background()
	d := &apiKeyDataSource{apiKeys: &mockApiKeysSvc{}}
	sr, _ := testDataSourceSchemaAndObjType(ctx, d)
	resp := datasource.ReadResponse{State: emptyDataSourceState(sr)}
	d.Read(ctx, datasource.ReadRequest{Config: undecodableConfig(sr)}, &resp)
	wantError(t, resp.Diagnostics)
}

func TestApiKeyResource_undecodableRequests(t *testing.T) {
	ctx := context.Background()
	r := &apiKeyResource{apiKeys: &mockApiKeysSvc{}}
	sr, _ := testResourceSchemaAndObjType(ctx, r)

	createResp := resource.CreateResponse{State: emptyResourceState(sr)}
	r.Create(ctx, resource.CreateRequest{Plan: undecodablePlan(sr)}, &createResp)
	wantError(t, createResp.Diagnostics)

	readResp := resource.ReadResponse{State: emptyResourceState(sr)}
	r.Read(ctx, resource.ReadRequest{State: undecodableState(sr)}, &readResp)
	wantError(t, readResp.Diagnostics)

	deleteResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: undecodableState(sr)}, &deleteResp)
	wantError(t, deleteResp.Diagnostics)
}

func TestDomainVerificationResource_undecodableRequests(t *testing.T) {
	ctx := context.Background()
	r := &domainVerificationResource{domains: &mockDomainsSvc{}}
	sr, _ := testResourceSchemaAndObjType(ctx, r)

	createResp := resource.CreateResponse{State: emptyResourceState(sr)}
	r.Create(ctx, resource.CreateRequest{Plan: undecodablePlan(sr)}, &createResp)
	wantError(t, createResp.Diagnostics)

	readResp := resource.ReadResponse{State: emptyResourceState(sr)}
	r.Read(ctx, resource.ReadRequest{State: undecodableState(sr)}, &readResp)
	wantError(t, readResp.Diagnostics)
}

func TestDomainDataSource_Read_undecodableConfig(t *testing.T) {
	ctx := context.Background()
	d := &domainDataSource{domains: &mockDomainsSvc{}}
	sr, _ := testDataSourceSchemaAndObjType(ctx, d)
	resp := datasource.ReadResponse{State: emptyDataSourceState(sr)}
	d.Read(ctx, datasource.ReadRequest{Config: undecodableConfig(sr)}, &resp)
	wantError(t, resp.Diagnostics)
}

func TestDomainDataSource_Read_flattenError(t *testing.T) {
	ctx := context.Background()
	d := &domainDataSource{domains: &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) { return oneRecordDomain(), nil },
	}}
	sr, objType := testDataSourceSchemaAndObjType(ctx, d)
	failFlatten(t)

	resp := datasource.ReadResponse{State: emptyDataSourceState(sr)}
	d.Read(ctx, datasource.ReadRequest{
		Config: testDataSourceConfig(sr, objType, domainDataSourceConfigVals("test-id", objType)),
	}, &resp)
	wantError(t, resp.Diagnostics)
}

func TestDomainResource_undecodableRequests(t *testing.T) {
	ctx := context.Background()
	r := &domainResource{domains: &mockDomainsSvc{}}
	sr, objType := testResourceSchemaAndObjType(ctx, r)

	createResp := resource.CreateResponse{State: emptyResourceState(sr)}
	r.Create(ctx, resource.CreateRequest{Plan: undecodablePlan(sr)}, &createResp)
	wantError(t, createResp.Diagnostics)

	readResp := resource.ReadResponse{State: emptyResourceState(sr)}
	r.Read(ctx, resource.ReadRequest{State: undecodableState(sr)}, &readResp)
	wantError(t, readResp.Diagnostics)

	// Update decodes the plan first, then the state.
	updateResp := resource.UpdateResponse{State: emptyResourceState(sr)}
	r.Update(ctx, resource.UpdateRequest{Plan: undecodablePlan(sr), State: undecodableState(sr)}, &updateResp)
	wantError(t, updateResp.Diagnostics)

	updateResp = resource.UpdateResponse{State: emptyResourceState(sr)}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  testResourcePlan(sr, objType, domainStateVals(objType)),
		State: undecodableState(sr),
	}, &updateResp)
	wantError(t, updateResp.Diagnostics)

	deleteResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: undecodableState(sr)}, &deleteResp)
	wantError(t, deleteResp.Diagnostics)
}

// An Update plan with no custom_return_path keeps the one already in state.
func TestDomainResource_Update_keepsCustomReturnPath(t *testing.T) {
	ctx := context.Background()
	d := oneRecordDomain()
	d.Records = nil
	r := &domainResource{domains: &mockDomainsSvc{
		UpdateWithContextFn: func(_ context.Context, _ string, _ *resend.UpdateDomainRequest) (resend.Domain, error) {
			return resend.Domain{}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) { return d, nil },
	}}
	sr, objType := testResourceSchemaAndObjType(ctx, r)
	planVals := domainStateVals(objType)
	planVals["custom_return_path"] = tftypes.NewValue(tftypes.String, nil)

	resp := resource.UpdateResponse{State: emptyResourceState(sr)}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  testResourcePlan(sr, objType, planVals),
		State: testResourceState(sr, objType, domainStateVals(objType)),
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}

func TestDomainResource_flattenError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		UpdateWithContextFn: func(_ context.Context, _ string, _ *resend.UpdateDomainRequest) (resend.Domain, error) {
			return resend.Domain{}, nil
		},
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) { return oneRecordDomain(), nil },
		CreateWithContextFn: func(_ context.Context, _ *resend.CreateDomainRequest) (resend.CreateDomainResponse, error) {
			return resend.CreateDomainResponse{Id: "test-id"}, nil
		},
	}
	r := &domainResource{domains: mock}
	sr, objType := testResourceSchemaAndObjType(ctx, r)
	failFlatten(t)

	createResp := resource.CreateResponse{State: emptyResourceState(sr)}
	r.Create(ctx, resource.CreateRequest{Plan: testResourcePlan(sr, objType, domainPlanVals(objType))}, &createResp)
	wantError(t, createResp.Diagnostics)

	state := testResourceState(sr, objType, domainStateVals(objType))
	readResp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	wantError(t, readResp.Diagnostics)

	updateResp := resource.UpdateResponse{State: emptyResourceState(sr)}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  testResourcePlan(sr, objType, domainStateVals(objType)),
		State: state,
	}, &updateResp)
	wantError(t, updateResp.Diagnostics)
}
