package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// readOptionalString converts a *string from a GraphQL response into a
// types.String, mapping both nil and "" to null. Used for API fields that
// are Optional in the resource schema and where the server may respond with
// either null or an empty string.
func readOptionalString(v *string) types.String {
	if v == nil || *v == "" {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

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
// processing request" when creating a volume on a newly created service) or
// per-mutation creation throttles (see isCreationRateLimited).
func retryCreateContext(ctx context.Context, timeout time.Duration, f func() error) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		err := f()
		if err == nil {
			return nil
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "problem processing request") ||
			isRateLimited(err) ||
			isCreationRateLimited(err) {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
}

// isCreationRateLimited returns true if the error indicates a Railway
// per-mutation creation throttle. Distinct from the account-wide isRateLimited
// / isCreationCooldown patterns — this fires when the SAME mutation is called
// too rapidly (e.g. two volumeCreate calls within seconds) and is answered with
// Railway's characteristic "Whoa there pal!" wording. Transient; a short wait
// clears it.
func isCreationRateLimited(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "whoa there pal") ||
		strings.Contains(msg, "try again in a sec")
}

// retryReadAfterCreateContext retries a post-create read that may fail due to
// Railway's eventual consistency: a resource that was just created can take
// several seconds to appear in list/query endpoints. Unlike retryCreateContext,
// this treats "not found" as retryable — the resource DOES exist server-side,
// the API just hasn't propagated it to the read path yet. Also retries on rate
// limits.
func retryReadAfterCreateContext(ctx context.Context, timeout time.Duration, f func() error) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		err := f()
		if err == nil {
			return nil
		}
		if isNotFound(err) || isRateLimited(err) {
			return retry.RetryableError(err)
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "problem processing request") {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
}

// retryUpdateContext retries a volume update mutation (rename via
// updateVolume, or mount path move via updateVolumeInstance) against
// Railway's transient errors: the per-mutation throttle, or the
// eventual-consistency window where a mutation issued moments after create
// can see "not found" or "Problem processing request" while the object is
// still propagating. Used both for the post-create rename in Create (where
// an unretried failure here left a tainted, prevent_destroy-blocked resource
// behind) and for renames/mount-path changes in Update (where failures don't
// taint, but retrying still avoids a needless operator-visible error on a
// transient blip).
func retryUpdateContext(ctx context.Context, timeout time.Duration, f func() error) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		err := f()
		if err == nil {
			return nil
		}
		if isNotFound(err) || isRateLimited(err) {
			return retry.RetryableError(err)
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "problem processing request") {
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

// isCreationCooldown returns true if the error indicates one of Railway's
// per-workspace-or-user creation cooldowns:
//
//   - projectCreate  — "1 project per 30 seconds"
//   - environmentCreate — "Only one environment can be created per user every 30s"
//
// Both are recoverable by waiting and retrying.
func isCreationCooldown(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "creating projects too quickly") ||
		strings.Contains(msg, "1 project per 30 seconds") ||
		strings.Contains(msg, "only one environment can be created")
}

// retryOnCooldownContext wraps a create-side call, retrying if the error is
// one of Railway's known per-user / per-workspace creation cooldowns
// (project-per-30s, env-per-30s). Bounded by the supplied timeout — cooldowns
// max out at ~30s so 90s is a safe default that survives a couple of back-to-
// back tests without hanging on a genuine failure.
//
// The tofu-plugin-sdk retry helper uses its own polling cadence; there is no
// need to sleep between attempts here.
func retryOnCooldownContext(ctx context.Context, timeout time.Duration, f func() error) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		err := f()
		if err == nil {
			return nil
		}
		if isCreationCooldown(err) || isRateLimited(err) {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
}

// isRedeployNotReady returns true if the error indicates Railway's
// serviceInstanceRedeploy mutation was rejected because a previous deployment
// on the same service is still building. This happens on rapid apply cycles
// (initial deploy still in progress when a variable_collection or variable
// mutation triggers a follow-up redeploy) and is fully transient — waiting
// for the in-flight build to finish and retrying resolves it.
func isRedeployNotReady(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cannot redeploy yet") ||
		strings.Contains(msg, "wait for the original deployment")
}

// retryRedeployContext wraps a serviceInstanceRedeploy call, retrying if the
// error is Railway's "Cannot redeploy yet, please wait for the original
// deployment to finish building" transient conflict. Also retries on rate
// limits. Everything else is treated as non-retryable so genuine failures
// (auth, invalid service, etc.) surface immediately.
//
// Default budget: 3 minutes. Railway builds typically settle in under a
// minute; 3 minutes covers the tail with margin.
func retryRedeployContext(ctx context.Context, timeout time.Duration, f func() error) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		err := f()
		if err == nil {
			return nil
		}
		if isRedeployNotReady(err) || isRateLimited(err) {
			return retry.RetryableError(err)
		}
		return retry.NonRetryableError(err)
	})
}
