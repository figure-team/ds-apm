package dlq

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign returns a detached HMAC-SHA256 signature (hex-encoded) over payload
// using key. It is used to authenticate DLQ replay payloads so a forged or
// tampered re-delivery request can be rejected (FR-CF5.3 / NF-5.3.1).
func Sign(key, payload []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether signature is a valid HMAC-SHA256 signature over
// payload for key. The comparison is constant-time. A malformed (non-hex)
// or empty signature is rejected rather than causing a panic.
func Verify(key, payload []byte, signature string) bool {
	expected, err := hex.DecodeString(signature)
	if err != nil || len(expected) == 0 {
		return false
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	return hmac.Equal(expected, mac.Sum(nil))
}
