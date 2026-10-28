package provider_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	resend "github.com/resend/resend-go/v4"
)

func init() {
	resource.AddTestSweepers("resend_domain", &resource.Sweeper{
		Name: "resend_domain",
		F:    sweepDomains,
	})

	resource.AddTestSweepers("resend_api_key", &resource.Sweeper{
		Name:         "resend_api_key",
		F:            sweepApiKeys,
		Dependencies: []string{"resend_domain"},
	})
}

func sweepDomains(_ string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return nil
	}

	client := resend.NewClient(apiKey)
	ctx := context.Background()

	domains, err := client.Domains.ListWithContext(ctx)
	if err != nil {
		return err
	}

	for _, domain := range domains.Data {
		if strings.HasPrefix(domain.Name, "tf-test-") {
			if _, err := client.Domains.RemoveWithContext(ctx, domain.Id); err != nil {
				return err
			}
		}
	}

	return nil
}

func sweepApiKeys(_ string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return nil
	}

	client := resend.NewClient(apiKey)
	ctx := context.Background()

	keys, err := client.ApiKeys.ListWithContext(ctx)
	if err != nil {
		return err
	}

	for _, key := range keys.Data {
		if strings.HasPrefix(key.Name, "tf-test-") {
			if _, err := client.ApiKeys.RemoveWithContext(ctx, key.Id); err != nil {
				return err
			}
		}
	}

	return nil
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}
