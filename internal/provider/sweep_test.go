package provider_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	resend "github.com/resend/resend-go/v4"
)

// sweepPrefix is the name prefix every acceptance test gives its resources. It
// is the only thing a sweeper matches on, so nothing else in the account is
// ever deleted.
const sweepPrefix = "tf-test-"

// sweepPageSize is the largest page the Resend list endpoints return.
const sweepPageSize = 100

func init() {
	// Keys go first: a key can be scoped to a domain.
	resource.AddTestSweepers("resend_api_key", &resource.Sweeper{
		Name: "resend_api_key",
		F:    sweepApiKeys,
	})

	resource.AddTestSweepers("resend_domain", &resource.Sweeper{
		Name:         "resend_domain",
		F:            sweepDomains,
		Dependencies: []string{"resend_api_key"},
	})
}

// sweepItem is the part of a listed Resend object a sweeper needs.
type sweepItem struct {
	id   string
	name string
}

// sweepPage fetches the page after the given cursor (nil for the first page).
type sweepPage func(ctx context.Context, after *string) (items []sweepItem, hasMore bool, err error)

// sweepByPrefix deletes every listed object whose name starts with sweepPrefix. It
// collects all matches across pages before deleting any, so removing an object
// cannot disturb the pagination cursor.
func sweepByPrefix(ctx context.Context, page sweepPage, remove func(ctx context.Context, id string) error) error {
	var ids []string

	var after *string

	for {
		items, hasMore, err := page(ctx, after)
		if err != nil {
			return fmt.Errorf("listing objects to sweep: %w", err)
		}

		for _, item := range items {
			if strings.HasPrefix(item.name, sweepPrefix) {
				ids = append(ids, item.id)
			}
		}

		if !hasMore || len(items) == 0 {
			break
		}

		after = &items[len(items)-1].id
	}

	for _, id := range ids {
		if err := remove(ctx, id); err != nil {
			return fmt.Errorf("removing %s: %w", id, err)
		}
	}

	return nil
}

func sweepClient() *resend.Client {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return nil
	}

	return resend.NewClient(apiKey)
}

func sweepDomains(_ string) error {
	client := sweepClient()
	if client == nil {
		return nil
	}

	limit := sweepPageSize

	return sweepByPrefix(context.Background(),
		func(ctx context.Context, after *string) ([]sweepItem, bool, error) {
			resp, err := client.Domains.ListWithOptions(ctx, &resend.ListOptions{Limit: &limit, After: after})
			if err != nil {
				return nil, false, err
			}

			items := make([]sweepItem, 0, len(resp.Data))
			for _, d := range resp.Data {
				items = append(items, sweepItem{id: d.Id, name: d.Name})
			}

			return items, resp.HasMore, nil
		},
		func(ctx context.Context, id string) error {
			_, err := client.Domains.RemoveWithContext(ctx, id)

			return err
		})
}

func sweepApiKeys(_ string) error {
	client := sweepClient()
	if client == nil {
		return nil
	}

	limit := sweepPageSize

	return sweepByPrefix(context.Background(),
		func(ctx context.Context, after *string) ([]sweepItem, bool, error) {
			resp, err := client.ApiKeys.ListWithOptions(ctx, &resend.ListOptions{Limit: &limit, After: after})
			if err != nil {
				return nil, false, err
			}

			items := make([]sweepItem, 0, len(resp.Data))
			for _, k := range resp.Data {
				items = append(items, sweepItem{id: k.Id, name: k.Name})
			}

			return items, resp.HasMore, nil
		},
		func(ctx context.Context, id string) error {
			_, err := client.ApiKeys.RemoveWithContext(ctx, id)

			return err
		})
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}
