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

// isNotFoundOrGone is like isNotFound but also matches Railway's non-standard
// responses for already-deleted resources ("Not Authorized", "Problem processing
// request"). Use ONLY in Delete methods — never in Read, where a false positive
// would silently remove live resources from state.
func isNotFoundOrGone(err error) bool {
	if isNotFound(err) {
		return true
	}
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not authorized") ||
		strings.Contains(msg, "problem processing request")
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

// retryCreateContext retries a function that may return transient errors during
// resource creation due to Railway's eventual consistency (e.g., "Problem
// processing request" when creating a volume on a newly created service).
func retryCreateContext(ctx context.Context, timeout time.Duration, f func() error) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		err := f()
		if err == nil {
			return nil
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "problem processing request") || isRateLimited(err) {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
}

// isOperationInProgress returns true if the error indicates a Railway
// "operation is already in progress" conflict. This commonly occurs during
// concurrent deletes of domain/proxy resources.
func isOperationInProgress(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "operation is already in progress")
}

// isRateLimited returns true if the error indicates a Railway rate limit.
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "try again in")
}
