// pkg/types/alertmanagertypes/dlq.go
package alertmanagertypes

import "time"

// DLQEntry represents one failed notification delivery returned by the DLQ API.
type DLQEntry struct {
	EventID  string    `json:"event_id"`
	Channel  string    `json:"channel"`
	Payload  []byte    `json:"payload"` // base64 in JSON responses
	FailedAt time.Time `json:"failed_at"`
	Reason   string    `json:"reason"`
	Status   string    `json:"status"` // "pending" | "replayed" | "replay_failed"
}

// ReplayResult summarises the outcome of a ReplayDLQEntries call.
type ReplayResult struct {
	Replayed int `json:"replayed"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}
