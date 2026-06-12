package dlq

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestReadEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dlq.jsonl")
	sink, err := NewJSONLDeadLetterSink(path, DefaultJSONLDeadLetterMaxSizeBytes)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := sink.Write(&Entry{
			EventID:  fmt.Sprintf("evt-%d", i),
			Channel:  "slack",
			Payload:  []byte(`{"k":"v"}`),
			FailedAt: time.Now().UTC(),
			Reason:   "boom",
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
	// Entries are returned in write order.
	if entries[0].EventID != "evt-0" || entries[2].EventID != "evt-2" {
		t.Fatalf("entries out of order: %q ... %q", entries[0].EventID, entries[2].EventID)
	}
	if entries[0].Channel != "slack" {
		t.Fatalf("payload not preserved: %+v", entries[0])
	}
}

func TestReadEntriesMissingFileIsEmpty(t *testing.T) {
	entries, err := ReadEntries(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	if err != nil {
		t.Fatalf("a missing DLQ file must read as empty, not error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("want 0 entries, got %d", len(entries))
	}
}

func TestReadEntriesIncludesRotatedSiblings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dlq.jsonl")
	sink, err := NewJSONLDeadLetterSink(path, 64 /* tiny → force rotation */)
	if err != nil {
		t.Fatal(err)
	}
	const n = 5
	for i := 0; i < n; i++ {
		if err := sink.Write(&Entry{
			EventID:  fmt.Sprintf("evt-%d", i),
			Channel:  "webhook",
			Payload:  []byte(`{"text":"hello world padding"}`),
			FailedAt: time.Now().UTC(),
			Reason:   "timeout",
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != n {
		t.Fatalf("replay must include rotated siblings: want %d entries, got %d", n, len(entries))
	}
}
