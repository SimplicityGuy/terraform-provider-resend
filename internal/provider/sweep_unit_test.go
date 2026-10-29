package provider_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestSweepByPrefix_deletesOnlyPrefixedAcrossPages(t *testing.T) {
	pages := map[string]struct {
		items   []sweepItem
		hasMore bool
	}{
		"":  {[]sweepItem{{"a", "tf-test-1"}, {"b", "production"}}, true},
		"b": {[]sweepItem{{"c", "tf-test-2"}, {"d", "my-tf-test-x"}}, true},
		"d": {[]sweepItem{{"e", "tf-test-3"}}, false},
	}

	var removed []string

	err := sweepByPrefix(context.Background(),
		func(_ context.Context, after *string) ([]sweepItem, bool, error) {
			key := ""
			if after != nil {
				key = *after
			}

			p := pages[key]

			return p.items, p.hasMore, nil
		},
		func(_ context.Context, id string) error {
			removed = append(removed, id)

			return nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := []string{"a", "c", "e"}; !reflect.DeepEqual(removed, want) {
		t.Errorf("removed %v, want %v", removed, want)
	}
}

func TestSweepByPrefix_stopsOnEmptyPageEvenIfHasMore(t *testing.T) {
	calls := 0

	err := sweepByPrefix(context.Background(),
		func(context.Context, *string) ([]sweepItem, bool, error) {
			calls++

			return nil, true, nil
		},
		func(context.Context, string) error { return nil })
	if err != nil || calls != 1 {
		t.Errorf("err=%v calls=%d, want nil and 1", err, calls)
	}
}

func TestSweepByPrefix_listError(t *testing.T) {
	boom := errors.New("boom")

	err := sweepByPrefix(context.Background(),
		func(context.Context, *string) ([]sweepItem, bool, error) { return nil, false, boom },
		func(context.Context, string) error { t.Fatal("must not remove"); return nil })
	if !errors.Is(err, boom) {
		t.Errorf("got %v, want wrapped boom", err)
	}
}

func TestSweepByPrefix_removeError(t *testing.T) {
	boom := errors.New("boom")

	err := sweepByPrefix(context.Background(),
		func(context.Context, *string) ([]sweepItem, bool, error) {
			return []sweepItem{{"a", "tf-test-1"}}, false, nil
		},
		func(context.Context, string) error { return boom })
	if !errors.Is(err, boom) {
		t.Errorf("got %v, want wrapped boom", err)
	}
}

func TestSweepers_noopWithoutKey(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "")

	if err := sweepDomains(""); err != nil {
		t.Errorf("sweepDomains: %v", err)
	}

	if err := sweepApiKeys(""); err != nil {
		t.Errorf("sweepApiKeys: %v", err)
	}
}
