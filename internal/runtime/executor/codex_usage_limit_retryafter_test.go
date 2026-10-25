package executor

import (
	"net/http"
	"testing"
	"time"
)

// TestCodexUsageLimit429CarriesRetryAfter pins the resets_at precise-cooldown
// contract: a 429 usage_limit_reached body must keep its reset timing after
// newCodexStatusErrWithCooling, or the conductor cools the credential for the
// legacy 60s default instead of until the upstream reset point.
func TestCodexUsageLimit429CarriesRetryAfter(t *testing.T) {
	body := []byte(`{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","plan_type":"prolite","resets_at":1791141009,"eligible_promo":null,"limit_window_minutes":10080,"resets_in_seconds":179863}}`)
	err := newCodexStatusErrWithCooling(http.StatusTooManyRequests, body, false)

	got := err.RetryAfter()
	if got == nil {
		t.Fatalf("retryAfter = nil, want ~49h parsed from resets_at/resets_in_seconds")
	}
	if *got < 24*time.Hour {
		t.Fatalf("retryAfter = %v, want >= 24h (weekly reset)", *got)
	}
	if !err.IsCredentialScoped() {
		t.Fatalf("usage_limit_reached must be credential-scoped when model-level cooling is off")
	}
}
