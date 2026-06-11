package dlq

import (
	"path/filepath"
	"testing"
)

func TestReplayLedgerIsIdempotent(t *testing.T) {
	led, err := NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer led.Close()
	if !led.MarkIfNew("evt-1") {
		t.Fatal("first mark should succeed")
	}
	if led.MarkIfNew("evt-1") {
		t.Fatal("second mark should report not new")
	}
	if !led.MarkIfNew("evt-2") {
		t.Fatal("different event should succeed")
	}
}

func TestReplayLedgerSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	led1, err := NewReplayLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	if !led1.MarkIfNew("evt-1") {
		t.Fatal("first mark should succeed")
	}
	if err := led1.Close(); err != nil {
		t.Fatal(err)
	}
	led2, err := NewReplayLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer led2.Close()
	if led2.MarkIfNew("evt-1") {
		t.Fatal("reopened ledger should remember evt-1")
	}
	if !led2.MarkIfNew("evt-3") {
		t.Fatal("new event after reopen should succeed")
	}
}

// TestIdempotencyKey is the DoD row-2 acceptance test for the extended
// idempotency key: key = sha256(fingerprint‖channel‖round). Replaying the
// same (fingerprint, channel, round) must collapse to the same key — so the
// ledger skips it as already-seen — while bumping the round produces a fresh
// key the ledger accepts as new. Changing the fingerprint or channel must
// also produce distinct keys so cross-channel replays never alias.
func TestIdempotencyKey(t *testing.T) {
	const fp = "abc123fingerprint"
	const channel = "slack"

	// Deterministic: identical (fp, channel, round) → identical key.
	if IdempotencyKey(fp, channel, 0) != IdempotencyKey(fp, channel, 0) {
		t.Fatal("IdempotencyKey must be deterministic for identical inputs")
	}

	// Bumping the round → a different key.
	if IdempotencyKey(fp, channel, 0) == IdempotencyKey(fp, channel, 1) {
		t.Fatal("incrementing round must change the key")
	}

	// Different channel → a different key (no cross-channel aliasing).
	if IdempotencyKey(fp, channel, 0) == IdempotencyKey(fp, "pagerduty", 0) {
		t.Fatal("different channel must change the key")
	}

	// Different fingerprint → a different key.
	if IdempotencyKey(fp, channel, 0) == IdempotencyKey("other-fp", channel, 0) {
		t.Fatal("different fingerprint must change the key")
	}

	// End-to-end with the ledger: same (fp,channel,round) is an idempotent
	// skip; bumping the round is a brand-new delivery.
	led, err := NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer led.Close()

	round0 := IdempotencyKey(fp, channel, 0)
	if !led.MarkIfNew(round0) {
		t.Fatal("first delivery of (fp,channel,round=0) must be new")
	}
	if led.MarkIfNew(IdempotencyKey(fp, channel, 0)) {
		t.Fatal("replaying the same (fp,channel,round=0) must be an idempotent skip")
	}
	if !led.MarkIfNew(IdempotencyKey(fp, channel, 1)) {
		t.Fatal("bumping the round must be accepted as a new delivery")
	}
}

func TestReplayLedgerRejectsEmptyEventID(t *testing.T) {
	led, err := NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer led.Close()
	if led.MarkIfNew("") {
		t.Fatal("empty event ID must not be marked new")
	}
}
