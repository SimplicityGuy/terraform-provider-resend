package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	resend "github.com/resend/resend-go/v4"
)

const (
	maxRetries    = 5
	baseBackoffMs = 1000
)

// handleDeleteError handles errors from delete operations.
// Returns true if the caller should return (either not-found or error added to diagnostics).
func handleDeleteError(err error, resourceType, id string, diagnostics *diag.Diagnostics) bool {
	if err == nil {
		return false
	}
	if isNotFoundError(err) {
		return true
	}
	diagnostics.AddError(
		"Error deleting "+resourceType,
		"Could not delete "+resourceType+" ID "+id+": "+err.Error(),
	)
	return true
}

// isNotFoundError checks if the error indicates a resource was not found.
// The Resend SDK returns generic error strings, so we check the message.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}

// isRetryableError checks if the error is retryable.
// This includes rate limit errors (HTTP 429) and transient server errors
// such as "Something went wrong" (HTTP 500) that the Resend API may return
// under load instead of a proper 429.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, resend.ErrRateLimit) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "something went wrong")
}

// after is the timer seam for retryOnRateLimit's backoff wait; tests replace it
// so they need not wait out real exponential backoff.
var after = time.After

// retryOnRateLimit retries the given function when a retryable error occurs,
// using exponential backoff. The Resend API allows 2 requests per second.
func retryOnRateLimit[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	var err error

	for attempt := range maxRetries {
		result, err = fn()
		if err == nil {
			return result, nil
		}

		if !isRetryableError(err) {
			return result, err
		}

		backoff := time.Duration(baseBackoffMs*(1<<attempt)) * time.Millisecond
		tflog.Debug(ctx, "Retryable error, retrying", map[string]any{
			"attempt": attempt + 1,
			"backoff": backoff.String(),
			"error":   err.Error(),
		})

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-after(backoff):
		}
	}

	return result, err
}
