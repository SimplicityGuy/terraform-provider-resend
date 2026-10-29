package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainResource_basic(t *testing.T) {
	domainName := testAccDomainName(t)
	resourceName := "resend_domain.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig_basic(domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", domainName),
					resource.TestCheckResourceAttr(resourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
			// Destroy is implicit
		},
	})
}

func TestAccDomainResource_update(t *testing.T) {
	domainName := testAccDomainName(t)
	resourceName := "resend_domain.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig_basic(domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", domainName),
					resource.TestCheckResourceAttr(resourceName, "open_tracking", "false"),
					resource.TestCheckResourceAttr(resourceName, "click_tracking", "false"),
					resource.TestCheckResourceAttr(resourceName, "tls", "opportunistic"),
				),
			},
			{
				Config: testAccDomainResourceConfig_tracking(domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", domainName),
					resource.TestCheckResourceAttr(resourceName, "open_tracking", "true"),
					resource.TestCheckResourceAttr(resourceName, "click_tracking", "true"),
					resource.TestCheckResourceAttr(resourceName, "tls", "enforced"),
				),
			},
		},
	})
}

func TestAccDomainResource_region(t *testing.T) {
	domainName := testAccDomainName(t)
	resourceName := "resend_domain.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig_region(domainName, "eu-west-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", domainName),
					resource.TestCheckResourceAttr(resourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
				),
			},
		},
	})
}

func TestAccDomainResource_import(t *testing.T) {
	domainName := testAccDomainName(t)
	resourceName := "resend_domain.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig_basic(domainName),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// These fields are not returned by the SDK Get endpoint
				ImportStateVerifyIgnore: []string{"open_tracking", "click_tracking", "tls", "custom_return_path"},
			},
		},
	})
}

func testAccDomainResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "resend_domain" "test" {
  name = %[1]q
}
`, name)
}

func testAccDomainResourceConfig_region(name, region string) string {
	return fmt.Sprintf(`
resource "resend_domain" "test" {
  name   = %[1]q
  region = %[2]q
}
`, name, region)
}

func testAccDomainResourceConfig_tracking(name string) string {
	return fmt.Sprintf(`
resource "resend_domain" "test" {
  name           = %[1]q
  open_tracking  = true
  click_tracking = true
  tls            = "enforced"
}
`, name)
}
