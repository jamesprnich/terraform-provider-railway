package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// NotFoundError indicates a resource was not found in the API.
// Used to distinguish "resource deleted externally" from real API errors.
type NotFoundError struct {
	ResourceType string
	Id           string
}

func (e *NotFoundError) Error() string {
	return e.ResourceType + " " + e.Id + " not found"
}

// isNotFound returns true if the error indicates a resource was not found.
// Checks for NotFoundError type and common Railway GraphQL API error patterns.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	var nfe *NotFoundError
	if errors.As(err, &nfe) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "could not find") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "not found")
}

// retryFindContext retries a function that may return NotFoundError due to
// eventual consistency. Non-NotFoundError errors are returned immediately.
func retryFindContext(ctx context.Context, timeout time.Duration, f func() error) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		err := f()
		if err == nil {
			return nil
		}
		if isNotFound(err) {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
}
