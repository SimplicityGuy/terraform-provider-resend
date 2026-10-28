package provider_test

import (
	"os"
	"testing"

	"github.com/SimplicityGuy/terraform-provider-resend/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"resend": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("RESEND_API_KEY") == "" {
		t.Fatal("RESEND_API_KEY must be set for acceptance tests")
	}
}
