package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/SimplicityGuy/terraform-provider-resend/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
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

// testAccDomainName returns a random test domain under RESEND_TEST_DOMAIN, the
// parent domain no other Resend account uses. Resend rejects a test domain under
// a parent that another account has registered, so domain tests skip without it.
func testAccDomainName(t *testing.T) string {
	t.Helper()
	domain := os.Getenv("RESEND_TEST_DOMAIN")
	if domain == "" {
		t.Skip("RESEND_TEST_DOMAIN is not set: domain acceptance tests need a parent domain no other Resend account uses")
	}
	return fmt.Sprintf("tf-test-%s.%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum), domain)
}
