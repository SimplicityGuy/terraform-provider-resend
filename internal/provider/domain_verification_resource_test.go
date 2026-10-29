package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainVerificationResource_basic(t *testing.T) {
	domainName := testAccDomainName(t)
	resourceName := "resend_domain_verification.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainVerificationResourceConfig(domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "domain_id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
				),
			},
		},
	})
}

func testAccDomainVerificationResourceConfig(domainName string) string {
	return fmt.Sprintf(`
resource "resend_domain" "test" {
  name = %[1]q
}

resource "resend_domain_verification" "test" {
  domain_id = resend_domain.test.id
}
`, domainName)
}
