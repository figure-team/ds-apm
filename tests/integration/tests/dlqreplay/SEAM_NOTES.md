# WT-dlq seam — integration-phase wiring for DLQ replay

WT-dlq delivers the DLQ reliability **backend** and durability, all unit-tested
(T0, `./verify.sh` green):

| DoD row | Test | Status |
|---|---|---|
| HMAC replay sign/verify | `dlq/hmac_test.go::TestVerify_TamperedRejected` | ✅ green |
| idempotency key `sha256(fp‖channel‖round)` | `dlq/ledger_test.go::TestIdempotencyKey` | ✅ green |
| DLQ wiring via `SIGNOZ_DLQ_PATH` | `alertmanagerserver/server_dlq_test.go::TestDLQWiring` | ✅ green |
| replay → idempotent skip (T1) | `dlqreplay/01_replay_idempotent.py` | ⛔ skip-guarded (this seam) |

The replay **route** cannot be mounted from this worktree: the route table and
DI live in shared seam files that WT-dlq must not edit. Three edits close the
seam; then remove the skip guard (or run with `SIGNOZ_DLQ_REPLAY_E2E=1`).

## What the backend already provides (owned, done)

- `pkg/ruler/signozruler/replay_handler.go`
  - `Replayer` — idempotent replay (send-then-mark-on-success) over a DLQ
    entry source + `dlq.ReplayLedger`.
  - `NewReplayDLQHandler(key []byte, replayer *Replayer) http.HandlerFunc` —
    HMAC-verifies the request body (`X-Signoz-DLQ-Signature`) then runs replay,
    returns `ReplayResult` JSON. Rejects unsigned/forged bodies with 401.
  - `Redeliverer` interface — the only missing production piece (see step 3).
- `pkg/alertmanager/alertmanagernotify/dlq`: `Sign`/`Verify`, `IdempotencyKey`,
  `ReadEntries`, `JSONLDeadLetterSink`, `ReplayLedger`.
- `pkg/alertmanager/alertmanagerserver/server.go`: sink resolved from
  `SIGNOZ_DLQ_PATH` and forwarded to every dispatcher.

## Integration-phase edits (shared seam files — NOT done here)

1. **Interface** — `pkg/ruler/handler.go`: add
   `ReplayDLQ(http.ResponseWriter, *http.Request)` to the `Handler` interface.

2. **Route mount** — `pkg/apiserver/signozapiserver/ruler.go` (mirror the
   existing `router.Handle(...)` blocks):
   ```go
   if err := router.Handle("/api/v2/ds/alerts/dlq/replay",
       handler.New(provider.authZ.EditAccess(provider.rulerHandler.ReplayDLQ), handler.OpenAPIDef{
           ID: "ReplayDLQ", Tags: []string{"rules"},
           Summary: "Replay dead-lettered notifications",
           SuccessStatusCode: http.StatusOK,
           ErrorStatusCodes: []int{http.StatusUnauthorized},
           SecuritySchemes: newSecuritySchemes(types.RoleEditor),
       })).Methods(http.MethodPost).GetError(); err != nil {
       return err
   }
   ```

3. **DI + production `Redeliverer`** — `pkg/ruler/signozruler/handler.go` +
   `provider.go`: give the handler a `*Replayer` built from
   - source: `func() ([]*dlq.Entry, error) { return dlq.ReadEntries(os.Getenv("SIGNOZ_DLQ_PATH")) }`
   - ledger: `dlq.NewReplayLedger(<SIGNOZ_DLQ_PATH>.ledger)`
   - redeliver: a `Redeliverer` that re-injects the entry's alert batch into the
     alertmanager (e.g. `PutAlerts`) so it is re-dispatched.
   Then add the thin method `func (h *handler) ReplayDLQ(rw, req) { NewReplayDLQHandler(h.replayKey, h.replayer)(rw, req) }`
   (may live in `replay_handler.go`).

## Deployment defaults (bootstrap seam — `cmd/community`, compose, helm)

- `SIGNOZ_DLQ_PATH` default: `var/ds-apm/alert-dlq.jsonl`
  (`alertmanagerserver.DefaultDLQPath`).
- `SIGNOZ_DLQ_REPLAY_KEY`: shared HMAC key for replay-trigger auth.

## Running the T1 test once mounted

```
SIGNOZ_DLQ_REPLAY_E2E=1 ./accept.sh
```
The signoz fixture must boot with `env_overrides={"SIGNOZ_DLQ_PATH": "...",
"SIGNOZ_DLQ_REPLAY_KEY": "integration-replay-key"}`.
