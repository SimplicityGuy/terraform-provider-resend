package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApiKeyResource_basic(t *testing.T) {
	keyName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	resourceName := "resend_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApiKeyResourceConfig_basic(keyName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", keyName),
					resource.TestCheckResourceAttr(resourceName, "permission", "full_access"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
				),
			},
		},
	})
}

func TestAccApiKeyResource_withDomain(t *testing.T) {
	keyName := fmt.Sprintf("tf-test-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	domainName := testAccDomainName(t)
	resourceName := "resend_api_key.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApiKeyResourceConfig_withDomain(keyName, domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", keyName),
					resource.TestCheckResourceAttr(resourceName, "permission", "sending_access"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckResourceAttrSet(resourceName, "domain_id"),
				),
			},
		},
	})
}

func testAccApiKeyResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "resend_api_key" "test" {
  name = %[1]q
}
`, name)
}

func testAccApiKeyResourceConfig_withDomain(name, domainName string) string {
	return fmt.Sprintf(`
resource "resend_domain" "test" {
  name = %[2]q
}

resource "resend_api_key" "test" {
  name       = %[1]q
  permission = "sending_access"
  domain_id  = resend_domain.test.id
}
`, name, domainName)
}
