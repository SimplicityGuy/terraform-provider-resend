package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	resend "github.com/resend/resend-go/v4"
)

func domainDataSourceConfigVals(id string, objType tftypes.Object) map[string]tftypes.Value {
	recordsType := objType.AttributeTypes["records"]
	return map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, id),
		"name":       tftypes.NewValue(tftypes.String, nil),
		"region":     tftypes.NewValue(tftypes.String, nil),
		"status":     tftypes.NewValue(tftypes.String, nil),
		"created_at": tftypes.NewValue(tftypes.String, nil),
		"records":    tftypes.NewValue(recordsType, nil),
	}
}

func TestDomainDataSource_Read_apiError(t *testing.T) {
	ctx := context.Background()
	mock := &mockDomainsSvc{
		GetWithContextFn: func(_ context.Context, _ string) (resend.Domain, error) {
			return resend.Domain{}, errors.New("api error")
		},
	}

	d := &domainDataSource{domains: mock}
	schemaResp, objType := testDataSourceSchemaAndObjType(ctx, d)

	req := datasource.ReadRequest{
		Config: testDataSourceConfig(schemaResp, objType, domainDataSourceConfigVals("test-id", objType)),
	}
	resp := datasource.ReadResponse{
		State: emptyDataSourceState(schemaResp),
	}

	d.Read(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error in diagnostics")
	}
}

func TestNewDomainDataSource(t *testing.T) {
	d := NewDomainDataSource()
	if d == nil {
		t.Error("expected non-nil data source")
	}
}

func TestDomainDataSource_Metadata(t *testing.T) {
	d := &domainDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "resend"}
	resp := datasource.MetadataResponse{}
	d.Metadata(context.Background(), req, &resp)
	if resp.TypeName != "resend_domain" {
		t.Errorf("expected type name 'resend_domain', got %q", resp.TypeName)
	}
}

func TestDomainDataSource_Configure_nil(t *testing.T) {
	d := &domainDataSource{}
	req := datasource.ConfigureRequest{ProviderData: nil}
	resp := datasource.ConfigureResponse{}
	d.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if d.domains != nil {
		t.Error("expected nil domains for nil provider data")
	}
}

func TestDomainDataSource_Configure_valid(t *testing.T) {
	client := resend.NewClient("test-key")
	d := &domainDataSource{}
	req := datasource.ConfigureRequest{ProviderData: client}
	resp := datasource.ConfigureResponse{}
	d.Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
	if d.domains == nil {
		t.Error("expected domains to be set")
	}
}

func TestDomainDataSourceModel_populateFromDomain(t *testing.T) {
	ctx := context.Background()
	domain := resend.Domain{
		Id:        "test-id",
		Name:      "test.com",
		Status:    "verified",
		Region:    "us-east-1",
		CreatedAt: "2024-01-01",
		Records:   []resend.Record{},
	}

	var model domainDataSourceModel
	diags := model.populateFromDomain(ctx, domain)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if model.ID.ValueString() != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", model.ID.ValueString())
	}
}

func TestDomainDataSource_Read_success(t *testing.T) {
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

	d := &domainDataSource{domains: mock}
	schemaResp, objType := testDataSourceSchemaAndObjType(ctx, d)

	req := datasource.ReadRequest{
		Config: testDataSourceConfig(schemaResp, objType, domainDataSourceConfigVals("test-id", objType)),
	}
	resp := datasource.ReadResponse{
		State: emptyDataSourceState(schemaResp),
	}

	d.Read(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
}
