package coderca

// signatureKeys is the fixed, ordered allowlist of alert-label keys that
// define an error's identity for dedup. It is deliberately COARSE — high
// cardinality labels (pod, instance, replica, …) are excluded so that a flood
// of near-identical errors collapses onto one signature (design §6.2).
var signatureKeys = []string{"alertname", "service.name", "severity", "error_class"}

// ErrorSignature builds a stable, coarse identity string from a fixed subset
// of alert labels. High-cardinality labels are ignored. Severity is
// case-normalized. The result is deterministic and independent of map order.
func ErrorSignature(labels map[string]string) string {
	// STUB — replaced in GREEN.
	return ""
}

// DedupKey derives the stable dedup key for (org, service, signature). Equal
// inputs always yield the same key; different inputs yield different keys.
func DedupKey(orgID, service, signature string) string {
	// STUB — replaced in GREEN.
	return ""
}
