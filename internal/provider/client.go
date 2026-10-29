package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	resend "github.com/resend/resend-go/v4"
)

// extractClient extracts the Resend client from provider data.
func extractClient(providerData any, diagnostics *diag.Diagnostics) *resend.Client {
	if providerData == nil {
		return nil
	}
	client, ok := providerData.(*resend.Client)
	if !ok {
		diagnostics.AddError("Unexpected Configure Type", "Expected *resend.Client")
		return nil
	}
	return client
}

// configureDomainsSvc extracts the Domains service interface from provider data.
func configureDomainsSvc(providerData any, diagnostics *diag.Diagnostics) resend.DomainsSvc {
	client := extractClient(providerData, diagnostics)
	if client == nil {
		return nil
	}
	return client.Domains
}

// configureApiKeysSvc extracts the ApiKeys service interface from provider data.
func configureApiKeysSvc(providerData any, diagnostics *diag.Diagnostics) resend.ApiKeysSvc {
	client := extractClient(providerData, diagnostics)
	if client == nil {
		return nil
	}
	return client.ApiKeys
}
