package coderca

// ShouldSampleAudit decides whether the n-th occurrence of a repeated event
// (a skip reason, a dedup hit) should emit an audit record. Auditing every
// occurrence would amplify writes under the flood the gates exist to survive
// (and the auditor drops-on-full anyway), so we sample: always the first
// occurrence, then every everyN-th (design §6.4).
//
//	occurrence: 1-based count of this event (1 = first seen).
//	everyN:     sampling stride; <=1 means "audit every occurrence".
func ShouldSampleAudit(occurrence, everyN int) bool {
	// STUB — replaced in GREEN.
	return false
}
