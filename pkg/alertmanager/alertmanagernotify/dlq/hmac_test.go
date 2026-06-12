package dlq

import "testing"

// TestVerify_TamperedRejected is the DoD row-1 acceptance test for HMAC
// replay signing (FR-CF5.3 / NF-5.3.1): a signature produced over a payload
// with a key must verify only for that exact (key, payload) pair. Any
// tampering of the payload, a wrong key, or a malformed signature must be
// rejected — and rejection must never panic.
func TestVerify_TamperedRejected(t *testing.T) {
	key := []byte("super-secret-replay-key")
	payload := []byte(`{"event_id":"evt-1","channel":"slack","round":0}`)

	sig := Sign(key, payload)
	if sig == "" {
		t.Fatal("Sign must produce a non-empty signature")
	}

	// Given a valid (key, payload, signature) → verification passes.
	if !Verify(key, payload, sig) {
		t.Fatal("valid signature must verify")
	}

	// When the payload is tampered → verification against the original
	// signature is rejected.
	tampered := []byte(`{"event_id":"evt-1","channel":"pagerduty","round":0}`)
	if Verify(key, tampered, sig) {
		t.Fatal("tampered payload must be rejected against the original signature")
	}

	// When the key is wrong → verification is rejected.
	if Verify([]byte("wrong-key"), payload, sig) {
		t.Fatal("wrong key must be rejected")
	}

	// A malformed (non-hex) signature must be rejected, not panic.
	if Verify(key, payload, "zz-not-a-hex-signature") {
		t.Fatal("malformed signature must be rejected")
	}

	// An empty signature must be rejected, not panic.
	if Verify(key, payload, "") {
		t.Fatal("empty signature must be rejected")
	}
}
