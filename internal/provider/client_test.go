package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	resend "github.com/resend/resend-go/v4"
)

func TestExtractClient_nil(t *testing.T) {
	var diagnostics diag.Diagnostics
	client := extractClient(nil, &diagnostics)
	if client != nil {
		t.Error("expected nil client for nil provider data")
	}
	if diagnostics.HasError() {
		t.Error("expected no errors for nil provider data")
	}
}

func TestExtractClient_validClient(t *testing.T) {
	expected := resend.NewClient("test-key")
	var diagnostics diag.Diagnostics
	client := extractClient(expected, &diagnostics)
	if client != expected {
		t.Error("expected returned client to match input")
	}
	if diagnostics.HasError() {
		t.Error("expected no errors for valid client")
	}
}

func TestExtractClient_wrongType(t *testing.T) {
	var diagnostics diag.Diagnostics
	client := extractClient("not-a-client", &diagnostics)
	if client != nil {
		t.Error("expected nil client for wrong type")
	}
	if !diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}
}

func TestConfigureDomainsSvc_nil(t *testing.T) {
	var diagnostics diag.Diagnostics
	svc := configureDomainsSvc(nil, &diagnostics)
	if svc != nil {
		t.Error("expected nil service for nil provider data")
	}
	if diagnostics.HasError() {
		t.Error("expected no errors for nil provider data")
	}
}

func TestConfigureDomainsSvc_validClient(t *testing.T) {
	client := resend.NewClient("test-key")
	var diagnostics diag.Diagnostics
	svc := configureDomainsSvc(client, &diagnostics)
	if svc == nil {
		t.Error("expected non-nil service")
	}
	if diagnostics.HasError() {
		t.Error("expected no errors for valid client")
	}
}

func TestConfigureApiKeysSvc_nil(t *testing.T) {
	var diagnostics diag.Diagnostics
	svc := configureApiKeysSvc(nil, &diagnostics)
	if svc != nil {
		t.Error("expected nil service for nil provider data")
	}
	if diagnostics.HasError() {
		t.Error("expected no errors for nil provider data")
	}
}

func TestConfigureApiKeysSvc_validClient(t *testing.T) {
	client := resend.NewClient("test-key")
	var diagnostics diag.Diagnostics
	svc := configureApiKeysSvc(client, &diagnostics)
	if svc == nil {
		t.Error("expected non-nil service")
	}
	if diagnostics.HasError() {
		t.Error("expected no errors for valid client")
	}
}
