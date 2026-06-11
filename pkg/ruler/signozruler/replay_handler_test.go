package signozruler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify/dlq"
)

// fakeRedeliverer records how many times it was asked to re-send and can be
// configured to fail the first failTimes calls (simulating a still-down
// channel) before succeeding.
type fakeRedeliverer struct {
	calls     int
	failTimes int
}

func (f *fakeRedeliverer) Redeliver(_ context.Context, _ *dlq.Entry) error {
	f.calls++
	if f.failTimes > 0 {
		f.failTimes--
		return fmt.Errorf("channel still unavailable")
	}
	return nil
}

// TestReplayer_RedeliversThenIdempotentSkip is the core of DoD row 4: the
// first replay re-sends every dead-lettered entry; a second replay of the
// same entries is an idempotent skip — no double-delivery.
func TestReplayer_RedeliversThenIdempotentSkip(t *testing.T) {
	entries := []*dlq.Entry{
		{EventID: "fp-1", Channel: "slack", Payload: []byte(`{"a":1}`)},
		{EventID: "fp-2", Channel: "slack", Payload: []byte(`{"a":2}`)},
	}
	source := func() ([]*dlq.Entry, error) { return entries, nil }
	ledger, err := dlq.NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	fake := &fakeRedeliverer{}

	r := NewReplayer(source, ledger, fake)

	res1, err := r.Replay(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res1.Resent != 2 || res1.Skipped != 0 || res1.Failed != 0 {
		t.Fatalf("round 1: want resent=2 skipped=0 failed=0, got %+v", res1)
	}
	if fake.calls != 2 {
		t.Fatalf("round 1: want 2 redeliveries, got %d", fake.calls)
	}

	res2, err := r.Replay(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res2.Resent != 0 || res2.Skipped != 2 {
		t.Fatalf("round 2: want resent=0 skipped=2 (idempotent), got %+v", res2)
	}
	if fake.calls != 2 {
		t.Fatalf("round 2 must not re-send: want 2 redeliveries total, got %d", fake.calls)
	}
}

// TestReplayer_FailedRedeliveryIsRetriable guards the send-then-mark ordering:
// a redelivery that errors must NOT be recorded in the ledger, so a later
// replay retries it rather than skipping a never-delivered notification.
func TestReplayer_FailedRedeliveryIsRetriable(t *testing.T) {
	entries := []*dlq.Entry{{EventID: "fp-1", Channel: "slack", Payload: []byte(`{}`)}}
	source := func() ([]*dlq.Entry, error) { return entries, nil }
	ledger, err := dlq.NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	fake := &fakeRedeliverer{failTimes: 1} // first attempt fails, then succeeds

	r := NewReplayer(source, ledger, fake)

	res1, err := r.Replay(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res1.Failed != 1 || res1.Resent != 0 {
		t.Fatalf("round 1: want failed=1 resent=0, got %+v", res1)
	}

	res2, err := r.Replay(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res2.Resent != 1 {
		t.Fatalf("round 2 must retry the previously-failed delivery: got %+v", res2)
	}
}

func newReplayer(t *testing.T, entries []*dlq.Entry, fake *fakeRedeliverer) *Replayer {
	t.Helper()
	ledger, err := dlq.NewReplayLedger(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ledger.Close() })
	return NewReplayer(func() ([]*dlq.Entry, error) { return entries, nil }, ledger, fake)
}

// TestReplayDLQHandler_AcceptsValidSignature: a correctly HMAC-signed replay
// trigger runs the replay and returns the status payload.
func TestReplayDLQHandler_AcceptsValidSignature(t *testing.T) {
	key := []byte("replay-key")
	entries := []*dlq.Entry{
		{EventID: "fp-1", Channel: "slack", Payload: []byte(`{}`)},
		{EventID: "fp-2", Channel: "slack", Payload: []byte(`{}`)},
	}
	fake := &fakeRedeliverer{}
	replayer := newReplayer(t, entries, fake)

	body := []byte(`{"trigger":"manual"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/alerts/dlq/replay", bytes.NewReader(body))
	req.Header.Set(ReplaySignatureHeader, dlq.Sign(key, body))
	rec := httptest.NewRecorder()

	NewReplayDLQHandler(key, replayer)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("valid signature: want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if fake.calls != 2 {
		t.Fatalf("valid signature must trigger replay: want 2 redeliveries, got %d", fake.calls)
	}
	var resp struct {
		Status string       `json:"status"`
		Data   ReplayResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not JSON: %v: %s", err, rec.Body.String())
	}
	if resp.Data.Resent != 2 || resp.Data.Total != 2 {
		t.Fatalf("status payload wrong: %+v", resp.Data)
	}
}

// TestReplayDLQHandler_RejectsTamperedSignature: a request whose body does
// not match its signature is rejected (401) and triggers no redelivery.
func TestReplayDLQHandler_RejectsTamperedSignature(t *testing.T) {
	key := []byte("replay-key")
	fake := &fakeRedeliverer{}
	replayer := newReplayer(t, []*dlq.Entry{{EventID: "fp-1", Channel: "slack"}}, fake)

	body := []byte(`{"trigger":"manual"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/alerts/dlq/replay", bytes.NewReader(body))
	// Sign a different body → the signature will not match the real body.
	req.Header.Set(ReplaySignatureHeader, dlq.Sign(key, []byte(`{"trigger":"forged"}`)))
	rec := httptest.NewRecorder()

	NewReplayDLQHandler(key, replayer)(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tampered signature: want 401, got %d (%s)", rec.Code, rec.Body.String())
	}
	if fake.calls != 0 {
		t.Fatalf("tampered request must not trigger any redelivery, got %d calls", fake.calls)
	}
}

// TestReplayDLQHandler_RejectsMissingSignature: a request with no signature
// header is rejected (401) and triggers no redelivery.
func TestReplayDLQHandler_RejectsMissingSignature(t *testing.T) {
	key := []byte("replay-key")
	fake := &fakeRedeliverer{}
	replayer := newReplayer(t, []*dlq.Entry{{EventID: "fp-1", Channel: "slack"}}, fake)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/alerts/dlq/replay", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()

	NewReplayDLQHandler(key, replayer)(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing signature: want 401, got %d", rec.Code)
	}
	if fake.calls != 0 {
		t.Fatalf("unsigned request must not trigger any redelivery, got %d calls", fake.calls)
	}
}
