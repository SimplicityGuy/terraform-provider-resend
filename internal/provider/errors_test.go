package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	resend "github.com/resend/resend-go/v3"
)

func TestHandleDeleteError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectReturn   bool
		expectHasError bool
	}{
		{
			name:           "nil error",
			err:            nil,
			expectReturn:   false,
			expectHasError: false,
		},
		{
			name:           "not found error",
			err:            errors.New("[ERROR]: not found"),
			expectReturn:   true,
			expectHasError: false,
		},
		{
			name:           "other error",
			err:            errors.New("[ERROR]: internal server error"),
			expectReturn:   true,
			expectHasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics
			result := handleDeleteError(tt.err, "test_resource", "test-id", &diagnostics)
			if result != tt.expectReturn {
				t.Errorf("handleDeleteError() = %v, want %v", result, tt.expectReturn)
			}
			if diagnostics.HasError() != tt.expectHasError {
				t.Errorf("diagnostics.HasError() = %v, want %v", diagnostics.HasError(), tt.expectHasError)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "not found lowercase",
			err:      errors.New("[ERROR]: not found"),
			expected: true,
		},
		{
			name:     "not_found with underscore",
			err:      errors.New("[ERROR]: not_found"),
			expected: true,
		},
		{
			name:     "Not Found mixed case",
			err:      errors.New("[ERROR]: Not Found"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("[ERROR]: rate limit exceeded"),
			expected: false,
		},
		{
			name:     "empty error",
			err:      errors.New(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("isNotFoundError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      &resend.RateLimitError{Message: "rate limited"},
			expected: true,
		},
		{
			name:     "something went wrong",
			err:      errors.New("[ERROR]: Something went wrong"),
			expected: true,
		},
		{
			name:     "something went wrong lowercase",
			err:      errors.New("[ERROR]: something went wrong"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("[ERROR]: invalid request"),
			expected: false,
		},
		{
			name:     "not found error",
			err:      errors.New("[ERROR]: not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryOnRateLimit_success(t *testing.T) {
	calls := 0
	result, err := retryOnRateLimit(context.Background(), func() (string, error) {
		calls++
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryOnRateLimit_nonRateLimitError(t *testing.T) {
	calls := 0
	_, err := retryOnRateLimit(context.Background(), func() (string, error) {
		calls++
		return "", errors.New("some other error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for non-rate-limit), got %d", calls)
	}
}

func TestRetryOnRateLimit_rateLimitThenSuccess(t *testing.T) {
	calls := 0
	result, err := retryOnRateLimit(context.Background(), func() (string, error) {
		calls++
		if calls == 1 {
			return "", &resend.RateLimitError{Message: "rate limited"}
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetryOnRateLimit_serverErrorThenSuccess(t *testing.T) {
	calls := 0
	result, err := retryOnRateLimit(context.Background(), func() (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("[ERROR]: Something went wrong")
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

// stubAfter replaces the retry timer seam with one that records the requested
// backoffs and fires immediately, restoring the real timer on cleanup.
func stubAfter(t *testing.T) *[]time.Duration {
	t.Helper()
	var got []time.Duration
	orig := after
	after = func(d time.Duration) <-chan time.Time {
		got = append(got, d)
		ch := make(chan time.Time, 1)
		ch <- time.Time{}
		return ch
	}
	t.Cleanup(func() { after = orig })
	return &got
}

func TestRetryOnRateLimit_maxRetriesExhausted(t *testing.T) {
	waits := stubAfter(t)
	calls := 0
	_, err := retryOnRateLimit(context.Background(), func() (string, error) {
		calls++
		return "", &resend.RateLimitError{Message: "rate limited"}
	})
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if calls != maxRetries {
		t.Errorf("expected %d calls, got %d", maxRetries, calls)
	}
	want := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	if len(*waits) != len(want) {
		t.Fatalf("expected %d backoff waits, got %v", len(want), *waits)
	}
	for i, d := range want {
		if (*waits)[i] != d {
			t.Errorf("backoff %d: expected %v, got %v", i, d, (*waits)[i])
		}
	}
}

func TestRetryOnRateLimit_contextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	orig := after
	t.Cleanup(func() { after = orig })
	// The timer never fires, so only cancellation can end the backoff wait.
	after = func(time.Duration) <-chan time.Time { return nil }

	calls := 0
	_, err := retryOnRateLimit(ctx, func() (string, error) {
		calls++
		cancel()
		return "", &resend.RateLimitError{Message: "rate limited"}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call before cancellation, got %d", calls)
	}
}
