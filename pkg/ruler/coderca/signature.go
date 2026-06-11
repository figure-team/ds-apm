package coderca

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// signatureKeys is the fixed, ordered allowlist of alert-label keys that
// define an error's identity for dedup. It is deliberately COARSE — high
// cardinality labels (pod, instance, replica, …) are excluded so that a flood
// of near-identical errors collapses onto one signature (design §6.2).
var signatureKeys = []string{"alertname", "service.name", "severity", "error_class"}

// ErrorSignature builds a stable, coarse identity string from a fixed subset
// of alert labels. High-cardinality labels are ignored. Severity is
// case-normalized. The result is deterministic and independent of map order
// (iteration walks the fixed signatureKeys slice, not the map).
func ErrorSignature(labels map[string]string) string {
	parts := make([]string, 0, len(signatureKeys))
	for _, key := range signatureKeys {
		val := strings.TrimSpace(labels[key])
		if val == "" {
			continue
		}
		if key == "severity" {
			val = strings.ToLower(val)
		}
		parts = append(parts, key+"="+val)
	}
	return strings.Join(parts, "|")
}

// DedupKey derives the stable dedup key for (org, service, signature). Equal
// inputs always yield the same key; different inputs yield different keys. A
// NUL separator keeps field boundaries unambiguous (org "1a"+svc "b" must not
// collide with org "1"+svc "ab"), preserving tenant isolation.
func DedupKey(orgID, service, signature string) string {
	sum := sha256.Sum256([]byte(orgID + "\x00" + service + "\x00" + signature))
	return hex.EncodeToString(sum[:])
}
