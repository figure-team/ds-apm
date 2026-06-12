package signozruler

import (
	"context"
	"io"
	"net/http"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify/dlq"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/render"
)

// ReplaySignatureHeader carries the hex HMAC-SHA256 signature of the raw
// request body, authenticating a replay trigger (FR-CF5.3 / NF-5.3.1).
const ReplaySignatureHeader = "X-Signoz-DLQ-Signature"

// replay_handler.go is the DLQ replay API *backend* for WT-dlq. It is
// deliberately self-contained: the replay logic, the redelivery seam, and the
// HTTP handler factory all live here so they can be unit-tested without the
// shared ruler wiring. Mounting the route and injecting a production
// Redeliverer are integration-phase steps that touch shared seam files
// (pkg/ruler/handler.go interface, pkg/apiserver/signozapiserver/ruler.go
// route table, pkg/ruler/signozruler/provider.go dependency injection) which
// this worktree must not edit — see
// tests/integration/tests/dlqreplay/SEAM_NOTES.md for the exact wiring.

// ReplayRound is the idempotency round assigned to replayed deliveries. The
// original (failed) delivery is conceptually round 0; a replay is round 1 so
// its idempotency key — sha256(fingerprint‖channel‖round) — differs from the
// original, while two replays of the same entry collapse to the same key and
// the second is an idempotent skip.
const ReplayRound = 1

// Redeliverer re-sends a single dead-lettered notification. The alertmanager
// supplies the production implementation at integration time; unit tests use
// a fake. Injecting it keeps the replay logic testable without the full
// notification pipeline.
type Redeliverer interface {
	Redeliver(ctx context.Context, entry *dlq.Entry) error
}

// ReplayResult summarizes one replay invocation. It is the status payload
// returned by the replay endpoint.
type ReplayResult struct {
	Total   int `json:"total"`
	Resent  int `json:"resent"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// Replayer re-delivers dead-lettered notifications idempotently. It reads
// pending entries from source, redelivers each that has not already been
// replayed, and records successful redeliveries in a durable ledger so a
// repeat invocation skips them instead of double-sending.
type Replayer struct {
	source    func() ([]*dlq.Entry, error)
	ledger    *dlq.ReplayLedger
	redeliver Redeliverer
}

// NewReplayer constructs a Replayer over the given entry source, idempotency
// ledger, and redelivery seam.
func NewReplayer(source func() ([]*dlq.Entry, error), ledger *dlq.ReplayLedger, redeliver Redeliverer) *Replayer {
	return &Replayer{source: source, ledger: ledger, redeliver: redeliver}
}

// Replay re-delivers every pending dead-letter entry that has not already
// been successfully replayed.
//
// Ordering is send-then-mark: an entry is recorded in the ledger only after
// its redelivery succeeds. A redelivery that errors is left unmarked so a
// later replay retries it (at-least-once / eventual delivery), while an entry
// that already succeeded is skipped (no double-send). The idempotency key is
// sha256(fingerprint‖channel‖ReplayRound), so the same entry always maps to
// the same ledger key across replay invocations.
func (r *Replayer) Replay(ctx context.Context) (ReplayResult, error) {
	entries, err := r.source()
	if err != nil {
		return ReplayResult{}, err
	}

	res := ReplayResult{Total: len(entries)}
	for _, e := range entries {
		key := dlq.IdempotencyKey(e.EventID, e.Channel, ReplayRound)
		if r.ledger.Has(key) {
			res.Skipped++
			continue
		}
		if err := r.redeliver.Redeliver(ctx, e); err != nil {
			// Leave the key unmarked so a subsequent replay retries it.
			res.Failed++
			continue
		}
		// Record only on success. MarkIfNew is durable, so a crash after this
		// point still skips the entry on the next replay.
		r.ledger.MarkIfNew(key)
		res.Resent++
	}
	return res, nil
}

// NewReplayDLQHandler returns an http.HandlerFunc that triggers a DLQ replay
// after verifying the request body is HMAC-signed with key. An unsigned,
// wrongly signed, or tampered request is rejected (401) before any redelivery
// occurs.
//
func NewReplayDLQHandler(key []byte, replayer *Replayer) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "failed to read replay request body"))
			return
		}
		defer req.Body.Close() //nolint:errcheck

		// Authenticate the trigger before doing any work: the signature must
		// cover the exact request body with the shared key. This rejects
		// forged or tampered replay requests (FR-CF5.3 / NF-5.3.1).
		if !dlq.Verify(key, body, req.Header.Get(ReplaySignatureHeader)) {
			render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "invalid or missing replay signature"))
			return
		}

		res, err := replayer.Replay(req.Context())
		if err != nil {
			render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "dlq replay failed"))
			return
		}
		render.Success(rw, http.StatusOK, res)
	}
}
