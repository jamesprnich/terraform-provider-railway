package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestIsNotFound_wrappedNotFoundError proves that fmt.Errorf(...: %w, &NotFoundError{...})
// is correctly recognised by isNotFound via errors.As. This underlies the
// inline-volume readback retry: the "not yet visible" sentinel is a
// descriptive-message wrapper around a NotFoundError, and misclassifying it
// causes retryReadAfterCreateContext to bail in one poll interval instead of
// waiting out Railway's post-create eventual-consistency window.
func TestIsNotFound_wrappedNotFoundError(t *testing.T) {
	inner := &NotFoundError{ResourceType: "inline volume for service", Id: "svc-123"}
	wrapped := fmt.Errorf("inline volume not yet visible for service svc-123 in environment env-456: %w", inner)

	if !isNotFound(wrapped) {
		t.Fatalf("isNotFound(wrapped) = false, want true — the retry loop will misclassify this as terminal")
	}

	// Sanity: unwrap chain reaches the sentinel.
	var target *NotFoundError
	if !errors.As(wrapped, &target) {
		t.Fatalf("errors.As did not unwrap to *NotFoundError")
	}
	if target.ResourceType != "inline volume for service" || target.Id != "svc-123" {
		t.Fatalf("unwrapped sentinel had wrong fields: got %+v", target)
	}
}

// TestIsNotFound_barePlainStringDoesNotMatch is the negative case: a plain
// error whose message does NOT contain any of the "not found" substrings must
// return false. This is what the original bug looked like — a fmt.Errorf with
// "not yet visible" text and no wrapped sentinel classified as non-retryable.
func TestIsNotFound_barePlainStringDoesNotMatch(t *testing.T) {
	err := fmt.Errorf("inline volume not yet visible for service svc-123 in environment env-456")
	if isNotFound(err) {
		t.Fatalf("isNotFound(plain-string) = true, want false — this is what the pre-fix code returned")
	}
}

// TestRetryReadAfterCreateContext_retriesOnWrappedNotFound proves the retry
// helper actually retries when the callback returns a wrapped NotFoundError.
// Without this, the readback that motivated the fix would bail on the first
// call and never see the eventual-consistency window close.
func TestRetryReadAfterCreateContext_retriesOnWrappedNotFound(t *testing.T) {
	attempts := 0
	err := retryReadAfterCreateContext(context.Background(), 5*time.Second, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("inline volume not yet visible for service svc-123 in environment env-456: %w",
				&NotFoundError{ResourceType: "inline volume for service", Id: "svc-123"})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryReadAfterCreateContext returned err=%v, want nil after 3 attempts", err)
	}
	if attempts < 3 {
		t.Fatalf("retryReadAfterCreateContext called f() only %d times, want >=3 — the retry loop did not wait for eventual consistency", attempts)
	}
}

// TestRetryReadAfterCreateContext_barePlainStringDoesNotRetry pins the
// pre-fix behaviour so a future refactor that reintroduces a plain
// fmt.Errorf(...) without wrapping cannot silently break the retry loop again.
func TestRetryReadAfterCreateContext_barePlainStringDoesNotRetry(t *testing.T) {
	attempts := 0
	start := time.Now()
	err := retryReadAfterCreateContext(context.Background(), 5*time.Second, func() error {
		attempts++
		return fmt.Errorf("inline volume not yet visible for service svc-123 in environment env-456")
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("retryReadAfterCreateContext returned nil, want error — plain-string sentinel is not retryable")
	}
	if !strings.Contains(err.Error(), "not yet visible") {
		t.Fatalf("returned error did not preserve message: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("retryReadAfterCreateContext called f() %d times, want 1 — plain-string sentinel must NOT retry", attempts)
	}
	// The whole call must complete well inside the 5s budget — it should
	// bail immediately, not wait even a single poll interval.
	if elapsed > time.Second {
		t.Fatalf("retryReadAfterCreateContext took %v, want <1s — non-retryable errors must bail immediately", elapsed)
	}
}

// TestIsRedeployNotReady_matchesRailwayError proves the classifier fires on
// Railway's actual error strings for the "in-flight build blocks redeploy"
// case. Both the short form and the long form must match.
func TestIsRedeployNotReady_matchesRailwayError(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"canonical", "input:3: serviceInstanceRedeploy Cannot redeploy yet, please wait for the original deployment to finish building, then try again."},
		{"short", "Cannot redeploy yet"},
		{"substring", "please wait for the original deployment"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !isRedeployNotReady(fmt.Errorf("%s", c.msg)) {
				t.Fatalf("isRedeployNotReady returned false for %q — retry helper will misclassify Railway's transient conflict as terminal", c.msg)
			}
		})
	}
}

// TestIsRedeployNotReady_negative asserts the classifier does NOT fire on
// unrelated errors — anything else must remain a hard failure so genuine
// bugs surface immediately.
func TestIsRedeployNotReady_negative(t *testing.T) {
	cases := []string{
		"unauthorized",
		"service not found",
		"internal server error",
		"",
	}
	for _, msg := range cases {
		if isRedeployNotReady(fmt.Errorf("%s", msg)) {
			t.Fatalf("isRedeployNotReady returned true for %q — must not retry unrelated errors", msg)
		}
	}
	if isRedeployNotReady(nil) {
		t.Fatalf("isRedeployNotReady(nil) returned true, want false")
	}
}

// TestRetryRedeployContext_retriesUntilBuildFinishes proves the retry helper
// actually iterates on the transient conflict — the mock returns the sentinel
// twice, then succeeds. Without the helper the redeploy would fail on the
// first call in the pre-fix code path.
func TestRetryRedeployContext_retriesUntilBuildFinishes(t *testing.T) {
	attempts := 0
	err := retryRedeployContext(context.Background(), 5*time.Second, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("input:3: serviceInstanceRedeploy Cannot redeploy yet, please wait for the original deployment to finish building")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryRedeployContext returned err=%v, want nil after 3 attempts", err)
	}
	if attempts < 3 {
		t.Fatalf("retryRedeployContext called f() only %d times, want >=3 — retry loop did not wait out the in-flight build", attempts)
	}
}

// TestRetryRedeployContext_unrelatedErrorBailsImmediately pins that a
// non-transient error skips retry — e.g. an auth error should surface on
// the first call, not eat the 3-minute budget.
func TestRetryRedeployContext_unrelatedErrorBailsImmediately(t *testing.T) {
	attempts := 0
	start := time.Now()
	err := retryRedeployContext(context.Background(), 5*time.Second, func() error {
		attempts++
		return fmt.Errorf("unauthorized: bad token")
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("retryRedeployContext returned nil, want error")
	}
	if attempts != 1 {
		t.Fatalf("retryRedeployContext called f() %d times, want 1", attempts)
	}
	if elapsed > time.Second {
		t.Fatalf("retryRedeployContext took %v, want <1s — non-retryable errors must bail immediately", elapsed)
	}
}

// TestIsCreationRateLimited_matchesWhoaThere proves the classifier fires on
// Railway's per-mutation throttle wording. The characteristic phrase is
// "Whoa there pal!" — case-insensitive matching so the classifier is robust
// to Railway rephrasing it.
func TestIsCreationRateLimited_matchesWhoaThere(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"canonical", "input:3: volumeCreate Whoa there pal! You are creating volumes too quickly. Try again in a sec"},
		{"short", "Whoa there pal!"},
		{"casing", "WHOA THERE PAL"},
		{"try_again_substring", "please try again in a sec"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !isCreationRateLimited(fmt.Errorf("%s", c.msg)) {
				t.Fatalf("isCreationRateLimited returned false for %q — retryCreateContext will misclassify Railway's throttle as terminal", c.msg)
			}
		})
	}
}

// TestIsCreationRateLimited_negative asserts unrelated errors are NOT
// classified as retryable — otherwise a genuine failure would eat the retry
// budget instead of surfacing immediately.
func TestIsCreationRateLimited_negative(t *testing.T) {
	cases := []string{
		"unauthorized",
		"service not found",
		"internal server error",
		"",
	}
	for _, msg := range cases {
		if isCreationRateLimited(fmt.Errorf("%s", msg)) {
			t.Fatalf("isCreationRateLimited returned true for %q — must not retry unrelated errors", msg)
		}
	}
	if isCreationRateLimited(nil) {
		t.Fatalf("isCreationRateLimited(nil) returned true, want false")
	}
}

// TestRetryCreateContext_retriesOnCreationRateLimit proves the retry helper
// picks up the per-mutation throttle sentinel. Without this, a rapid parallel
// createVolume in the same apply graph would fail on the first call rather
// than waiting out Railway's few-second cool-off.
func TestRetryCreateContext_retriesOnCreationRateLimit(t *testing.T) {
	attempts := 0
	err := retryCreateContext(context.Background(), 5*time.Second, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("input:3: volumeCreate Whoa there pal! You are creating volumes too quickly")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryCreateContext returned err=%v, want nil after 3 attempts", err)
	}
	if attempts < 3 {
		t.Fatalf("retryCreateContext called f() only %d times, want >=3 — retry loop did not wait out the throttle", attempts)
	}
}
