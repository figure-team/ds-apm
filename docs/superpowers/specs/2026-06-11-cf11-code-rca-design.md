# CF-11 — AI Codebase RCA · Phase 0 Design

> **Status:** Phase 0 design (design-first; no implementation until approved + hardened).
> **Worktree:** `feat/cf11-code-rca` · **Scope authority:** [`SCOPE.md`](../../../SCOPE.md), [`TESTING.md`](../../../TESTING.md)
> **Date:** 2026-06-11

## 1. Goal & problem

When an alert fires that has **no matching SOP** (CF-1 "unbound") **and** there is an
anomaly/spike (CF-7), there is no human-authored runbook to follow. CF-11 closes that gap:
drive a **CLI coding agent** (`claude` / `codex`) to **explore the service's source code**
and produce a **root cause + proposed fix**, delivered to a human for review (HITL — never
auto-applied).

The hard problem is **not** the AI call; it is **cost/volume containment**. Hundreds of
near-identical errors each triggering a minutes-long, repo-exploring agent run = token and
process blow-up. This project has a documented quota-runaway history (`y2i` masking,
2026-04-26), so **cost control is the #1 design driver**, gating everything else.

## 2. Resolved Phase-0 decisions (user-confirmed 2026-06-11)

| # | SCOPE decision | Resolution |
|---|---|---|
| 1 | Cost/volume control | **Run-count + concurrency + dedup** (no token introspection — CLI does not return token counts reliably). Layered gates, §6. |
| 2 | Source-state schema + sync policy | **On-demand fetch + cached bare clone; baseline commit pinned at trigger.** §8. |
| — | Error-context scope (v1) | **Minimal (labels/annotations/signature) + `EvidenceCollector` interface** so logs/traces can be added later without rework. §7. |
| 3 | service→repo mapping | Explicit mapping table `(org_id, service_name) → repo_id [+ subpath]`. §8. |
| 4 | Security sandboxing | Secretbox creds in-process only; per-run isolated checkout; read-only CLI tool scope. §9. |
| 5 | Trigger precision | Gate predicate combining feature-on ∧ unbound ∧ anomaly(pluggable) ∧ dedup ∧ budget ∧ mapped ∧ severity. §10. |

## 3. Two grounding findings (verified against code)

1. **CF-7 (anomaly/spike) is not implemented in this layer.** Alerts arrive from upstream
   rule evaluation; there is no spike-detection interface to depend on. → The anomaly input
   is modeled as an **injected `AnomalySignal` interface with a conservative stub default**;
   real wiring is deferred to a seam. CF-11 must be useful with CF-1 alone.
2. **The dispatch hook receives only `labels` + `annotations` — no logs/traces/signature.**
   `dispatchhook.Apply(ctx, orgID, incidentID, alertFingerprint, labels, annotations)`
   (`pkg/ruler/aigenerator/dispatchhook/hook.go:85`). `EvidenceRefs` is intentionally empty
   in v0.1; there is no evidence collector. → v1 RCA context is built from labels/annotations
   + a derived error signature; richer log/trace evidence comes later via `EvidenceCollector`.

## 4. Reused existing assets (read / additive only)

- **CLI adapters** — `pkg/ruler/aigenerator/llmaigenerator/{claudecli,codexcli}` implement
  `Provider.Complete(ctx, system, user) (string, error)`. They exec the CLI but **do not set
  `cmd.Dir`** and are tuned for short (≤15s) text-in/text-out strategy generation. They live
  in the **ai/grounding WT's area** → **treated read-only.** CF-11 builds its own
  `coderca/clirunner` that **reuses their env/credential-prep approach** (e.g. codex
  `CODEX_HOME`/`auth.json` materialization, claude `CLAUDE_CODE_OAUTH_TOKEN`) but adds
  `cmd.Dir = <checkout>`, read-only tool flags, and minutes-scale timeouts.
- **secretbox** — `pkg/ruler/aiconfigstore/secretbox` (AES-256-GCM; key from
  `DS_APM_AI_CONFIG_ENCRYPTION_KEY`). Reused verbatim for git read-credentials.
- **Domain/Storable + Bun store template** — `ai_config.go` / `storable_ai_config.go` /
  `ai_config_store.go` + `sqlaiconfigstore` + migration `079`. Cloned as the pattern for
  `codebase_config`. Latest migration is `080` → new migrations are `081+`, registered in
  `pkg/signoz/provider.go`.
- **Audit** — `auditor.Audit(ctx, audittypes.AuditEvent)` (fire-and-forget) for CF-6.
- **History/dedup primitive** — `(orgID, incidentID)` history key + alert fingerprint
  (`pkg/ruler/aihistorystore`), informing the dedup key.

## 5. Architecture

### 5.1 Execution model — asynchronous (forced)

Code RCA takes **minutes**; the dispatch hook is deliberately fast (default **1s**
`hook.go:26`, **5s** in production wiring `signoz.go:420`) and must stay additive /
non-blocking (returns annotations unchanged on the unbound branch today). Synchronous-in-hook
is therefore impossible. Design:

```
[dispatch path, SEAM]  binding.Status != Bound
        │
        ▼
  coderca.Trigger.Maybe(ctx, signal)   ── gate (§6,§10) ──▶ skip+record reason (no CLI)
        │ passes gate
        ▼
  enqueue persisted coderca_run (status=queued)   ◀── dedup/idempotency anchor
        │
        ▼  (hook returns immediately)
  ┌─────────────────────────────────────────────┐
  │ bounded worker pool (concurrency cap)         │
  │  1. claim run (status=running)                │
  │  2. resolve service→repo (§8)                 │
  │  3. ensure source: fetch + pin baseline (§8)  │
  │  4. build prompt from error context (§7)      │
  │  5. clirunner: CLI in checkout, read-only (§9)│
  │  6. parse cause+fix + baseline commit (§7)    │
  │  7. deliver via handoff (CF-3) + history      │
  │  8. audit (CF-6); status=done/failed          │
  └─────────────────────────────────────────────┘
```

**Alternatives rejected:** sync-in-hook (blocks notify path); fire-and-forget goroutine with
no persistence (no dedup, no restart-safety, no cost ceiling, no UI state).

### 5.2 Component → package map (all new, within exclusive boundary)

| SCOPE component | Package / file |
|---|---|
| 1. Codebase config (back-office) | `pkg/types/ruletypes/codebase_config.go`, `storable_codebase_config.go`, `codebase_config_store.go` (iface); impl `pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseconfigstore/`; migration `pkg/sqlmigration/081_add_ds_codebase_config.go` |
| 2. Source state manager | `pkg/ruler/coderca/sourcestate/` — pure core + `GitRunner` interface |
| 3. Trigger gate | `pkg/ruler/coderca/trigger/` — pure predicate + `AnomalySignal` iface |
| 4. RCA orchestrator | `pkg/ruler/coderca/engine/` — gate→queue→worker; `pkg/ruler/coderca/clirunner/` — CLI exec adapter |
| — Run store/queue | `pkg/ruler/coderca/runstore/` (iface + sql impl); `coderca_run` table in migration `081` |
| 5. Delivery & safety | reuse handoff/history + `Auditor.Audit`; HITL suggestion-only |
| HTTP | `pkg/ruler/signozruler/coderca_handler.go` (methods only; registration = seam §11) |
| Frontend | new settings page + new locale namespace (separate files; routing = seam §11) |
| Tests | `tests/integration/tests/coderca/**` |

## 6. Cost/volume control (#1) — atomic DB admission + DB-backed leases

> **Design correction (Codex r1, CRITICAL):** the gates must NOT be in-memory snapshot
> predicates that "check then enqueue" — that leaks via TOCTOU across goroutines/replicas
> and across restarts (this server runs the dispatch hook in-process, `signoz.go:418`, and
> creates per-group goroutines, `dispatcher.go:357`; multiple replicas each run a pool).
> **All volume/cost enforcement lives inside DB transactions backed by unique constraints**,
> reusing the existing transactional store (`sqlstore.RunInTxCtx`, `sqlstore.go:25` /
> `bun.go:31`) and the unique-index pattern already used by `078_add_ds_apm_stores.go:71`.

### 6.1 Cheap pre-checks (no DB write)
Evaluated first, purely in-memory; reject before touching the DB:
1. **Feature gate** — per-org/per-service ON/OFF from `codebase_config` (default **OFF**,
   opt-in). Off → no-op.
2. **Base predicate** (§10): unbound ∧ anomaly(fail-closed) ∧ severity ≥ threshold.
3. **service→repo resolvable** — unmapped → skip(`no_repo_mapping`).

### 6.2 Atomic admission — `Admit(ctx, signal) (admitted bool, reason)`
A single transaction (`RunInTxCtx`, verified `sqlstore.go:25` → `bun.go:31`, SQLite
`sqlitesqlstore/provider.go:92`) is the only path that creates a run. **Backend reality:
only SQLite is registered (`provider.go:103`, modernc); Postgres is future.** `RunInTxCtx`
passes opts straight to `BeginTx` with **no serializable default** (`bun.go:38`), so the
atomicity argument is made explicit per backend, **not** assumed:

- **SQLite (current):** writers hold a single global write lock — admission transactions
  serialize, so the steps below cannot interleave. No row-locking syntax needed.
- **Postgres (future):** each step that decides capacity/dedup must take a **row lock**
  (`SELECT … FOR UPDATE` on the specific dedup/budget/capacity row) **or** run at
  `SERIALIZABLE` isolation with bounded retry. The design locks the relevant single row, so
  the decision serializes on PG too. **`FOR UPDATE SKIP LOCKED` is NOT used** (unsupported on
  SQLite, and it is the wrong tool for a capacity gate — Codex r2 CRITICAL).

Steps (all in one tx):
1. **Dedup with a true sliding cooldown** *(Codex r2: bucketed `floor(now/window)` admitted
   twice across a bucket edge)*. One row per key: unique `(org_id, dedup_key)` in
   `coderca_admission`, carrying `last_admitted_at`, `hit_count`, `last_run_ref`. In the tx,
   lock/upsert that row and compare:
   - `now − last_admitted_at < cooldown_window` (default **6h**) → duplicate: `hit_count += 1`
     (the §6.4 counter), return skip(`deduped`) with `last_run_ref`. Sliding window — no edge
     double-admit.
   - else (new or cooled-down) → set `last_admitted_at = now`, proceed.
   Under SQLite the serialized writer guarantees exactly one caller wins; under PG the row
   lock does. Collapses hundreds of identical errors to one run.
2. **Per-org budget** — conditionally bump a per-(org, day) counter row (`coderca_budget`):
   `… SET used = used + 1 WHERE used < max_runs_per_day` (default **20/day**). Affected = 0 →
   over budget → roll back, skip(`budget_exhausted`). Token-free.
3. **Queue depth** — `count(*) … status='queued'` for the org; ≥ `max_queue_depth`
   (default **50**) → roll back, skip(`queue_full`).
4. **Insert run** — `coderca_run(status='queued', dedup_key, baseline_commit=NULL, …)` and
   **commit**; set `coderca_admission.last_run_ref = run_id`. Any failure rolls back all atomically.

The decision logic is unit-testable against a fake tx store; the atomicity guarantee (one
admit per key under N concurrent callers, on the real SQLite store) is a required T1 test.

### 6.3 DB-backed worker lease, claim, recovery (Codex r1 #2; hardened Codex r2)
The worker pool is **stateless**; all concurrency/ownership lives in the DB so it is correct
across restarts and multiple replicas **without leader election**.

- `coderca_run` carries: `claimed_by` (instance id), `lease_token` (random per-claim **fence**,
  Codex r2), `lease_until`, `heartbeat_at`, `attempts`, `max_attempts` (default 2).

- **Concurrency cap via a locked capacity row, NOT a bare count race** *(Codex r2 CRITICAL: a
  plain `count(running) < max` lets two PG claimers both observe 0 and commit two runs).* A
  single `coderca_capacity` row per scope (`org_id` or global) holds `running` and
  `max_concurrent_runs` (default **1**, max **2**). The claim transaction makes the capacity
  decision against that **one locked row**, which serializes all claimers:
  ```sql
  -- inside RunInTxCtx. Step A: take the capacity slot (locks the single capacity row).
  --   SQLite: serialized writer. Postgres: this conditional UPDATE locks the row.
  UPDATE coderca_capacity SET running = running + 1
    WHERE scope = ? AND running < max_concurrent_runs;     -- affected=0 → at cap, abort claim
  -- Step B: only if a slot was taken, pick the oldest eligible queued run and mark it.
  UPDATE coderca_run SET status='running', claimed_by=?, lease_token=?,
         lease_until=now()+lease_ttl, heartbeat_at=now(), attempts=attempts+1
    WHERE id = (SELECT id FROM coderca_run
                WHERE status='queued' AND (lease_until IS NULL OR lease_until < now())
                ORDER BY created_at ASC LIMIT 1);
  -- commit both, or roll back (releasing the slot) if no queued row was found.
  ```
  `running` is a **counter reconciled by the reaper** (below), so the cap binds CLI processes
  **globally**, not per-process.
- **Fenced capacity-decrement contract (Codex r3).** `running` is decremented **only inside the
  same transaction** that performs a real `running → terminal/requeued` transition, **only when
  the fenced update `… WHERE id=? AND lease_token=?` affected exactly one row**. A stale owner
  whose fenced update affects **zero** rows (its run was reaped and reclaimed) **must not**
  decrement — otherwise it undercounts the semaphore and lets an extra runner in. Decrement is
  thus idempotent w.r.t. the live owner and inert for stale owners.

- **Heartbeat** — the worker periodically extends `lease_until` while the CLI runs.
- **Reaper** — a periodic sweep requeues runs that are `status='running'` AND
  `lease_until < now()` (→ `queued`, or `failed` once `attempts ≥ max_attempts`), and
  **reconciles `coderca_capacity.running`** to the count of live-leased runs (correcting any
  drift from missed decrements). Recovers crashed/killed replicas.
- **Duplicate-work guarantee, stated honestly (Codex r2/r3).** What is *guaranteed*: **no
  duplicate DB finalization** — the worker finalizes with `UPDATE … WHERE id=? AND
  lease_token=?`, so a reaped-then-reclaimed run's original worker cannot overwrite the new
  owner's result. What is *bounded but not eliminated*: duplicate **CLI cost** on a hard parent
  kill (see §6.5). With the live parent, `lease_ttl ≥ run_timeout + grace` (and heartbeat
  extension) ensures the lease does not expire before §6.5 force-kills the CLI, so the reaper
  cannot start a second run while the first is alive. On a SIGKILLed parent, the §6.5
  process-group / parent-death / startup-orphan-sweep mechanisms bound the residual window to
  ≈`run_timeout`; the cost ceiling (`--max-budget-usd`, timeout) caps the blast radius.
- **Portability** — `sqlstore` abstraction; correct on SQLite (serialized writer, the live
  backend) and PG (row lock on the capacity/queued rows, or SERIALIZABLE+retry). No
  `SKIP LOCKED`.

### 6.4 Skip recording — counters + sampled audit (Codex r1, HIGH #3)
Persisting a row + audit event per rejection is itself write-amplification under the exact
flood we are surviving (and `auditor.Audit` drops-on-full, `auditor.go:18`, so unsampled
audit silently loses data). Instead:
- **Dedup hits** increment `coderca_admission.hit_count` (no new row).
- **Other skips** (budget/queue/no_mapping) increment a per-(org, reason, day) aggregate
  counter (`coderca_skip_stat`, upsert) — **one row per (org, reason, day)**, not per alert.
- **Audit is sampled** — first occurrence per (dedup_key/reason, window) plus rate-limited
  thereafter — never per-alert.

### 6.5 Per-run hard ceilings + subprocess lifetime
- Wall-clock `run_timeout` (default **5m**) via ctx-cancel + `cmd.WaitDelay`.
- CLI-level caps as flags (§9): claude `--max-budget-usd` (hard $ ceiling), turn/file limits.
- Exceed → kill, status=`timeout`; the lease+reaper reclaim the slot.
- **Subprocess lifetime — orphan prevention (Codex r3, HIGH).** `exec.CommandContext` +
  `WaitDelay` (as `claudecli.go:79` / `codexcli.go:155` use) only kills the child while the
  **parent is alive**; a SIGKILLed server would orphan the CLI and the reaper could later start
  a second one. `clirunner` therefore:
  1. starts the CLI in its **own process group** (`SysProcAttr{Setpgid: true}`) and kills the
     **whole group** on ctx-cancel/timeout/shutdown (not just the lead pid);
  2. sets **parent-death signal** (`Pdeathsig: SIGKILL`) so the OS kills the child if the
     parent dies;
  3. runs a **startup orphan sweep** — on boot, before claiming, scan `/proc/*/{cmdline,environ}`
     for a coderca-specific marker and kill only **stale/orphaned** marked processes (must NOT
     kill another healthy local replica's live CLI — match on this instance's prior-run markers).
  **Caveats (Codex r4, non-blocking, noted for impl):** `Pdeathsig` fires on the creating
  **thread's** death, not whole-process death, and covers the direct child, not daemonized
  grandchildren that scrub env — so the process-group kill + orphan sweep are the primary
  guards, with `Pdeathsig` as defense-in-depth. Residual hard-kill window is bounded to
  ≈`run_timeout` and capped by `--max-budget-usd`.
  This makes §6.3's "no duplicate DB finalization" the *guarantee* and duplicate CLI cost a
  *bounded, mitigated* risk — not an unqualified "no double-run" claim.

All thresholds live in `codebase_config` (per-org overridable) with the defaults above.

## 7. Error context & RCA output

**Input (v1):** `RCAContext{ orgID, service, severity, environment, fingerprint,
error_signature, labels, annotations }`. An `EvidenceCollector` interface is defined now:

```go
type EvidenceCollector interface {
    Collect(ctx context.Context, sig ErrorSignature) ([]ruletypes.AIEvidenceRef, error)
}
```

v1 ships a **`NoopEvidenceCollector`** (returns empty); a logs/traces collector is a later,
additive implementation. This honors "inject logs·traces·signature" by *defining the seam*
without taking on the SigNoz query-API surface in v1.

**Prompt:** a coderca-owned system prompt instructs the agent to (a) explore the checkout to
locate code paths matching the error signature, (b) hypothesize a root cause, (c) propose a
fix as a *suggestion* (diff sketch / steps), (d) state confidence + limitations, (e) **echo
the baseline commit it analyzed**. Output is parsed to:

```go
type RCAResult struct {
    BaselineCommit string
    RootCause      string
    ProposedFix    string   // suggestion only; never applied
    Confidence     string   // high|medium|low
    Limitations    string
    Raw            string   // full CLI output, retained for audit
}
```

Parser is pure (raw CLI text → `RCAResult`), table-tested with golden fixtures, tolerant of
malformed output (status=`unparseable`, raw retained).

## 8. Data model + source state (#2) + service→repo (#3)

All tables are `org_id`-scoped, follow the `ai_config` Domain/Storable + Bun pattern, and ship
in migration **`081`** (registered last in `pkg/signoz/provider.go`). Secrets via `secretbox`.

| Table | Key | Purpose |
|---|---|---|
| `codebase_repo` | (org_id, repo_id) | repo registration + source state |
| `codebase_service_map` | (org_id, service_name) | service→repo[+subpath] mapping |
| `codebase_config` | (org_id) | per-org feature on/off + cost thresholds (§6) |
| `coderca_run` | (org_id, run_id) | one RCA run: status, dedup_key, baseline_commit, **lease fields** `claimed_by`/`lease_token`/`lease_until`/`heartbeat_at`/`attempts` (§6.3), result_ref |
| `coderca_admission` | unique (org_id, dedup_key) | dedup linchpin: `last_admitted_at` (sliding cooldown), `hit_count`, `last_run_ref` (§6.2/6.4) |
| `coderca_budget` | (org_id, day) | atomic per-day run counter (§6.2) |
| `coderca_capacity` | (scope) | locked concurrency semaphore: `running`, `max_concurrent_runs` (§6.3) |
| `coderca_skip_stat` | (org_id, reason, day) | aggregated skip counters (§6.4) |

- **`codebase_repo`** — `git_url`, `default_branch`, `credential_ciphertext` (secretbox),
  `enabled`. **Source state** surfaced in UI: `branch_name`, `fetched`(bool),
  `baseline_commit`, `last_sync_at`, `last_sync_status`.
- **`codebase_service_map`** — `(org_id, service_name)` → `repo_id` [+ optional `subpath`
  for monorepos].
- **Sync (on-demand):** maintain one **cached bare clone** per `repo_id` under a configured
  base dir. At trigger: `git fetch` the bare clone → resolve target-branch HEAD →
  `baseline_commit`; create a **per-run throwaway worktree/checkout** at that commit for the
  agent. `baseline_commit` is written to the `coderca_run` row and **echoed in `RCAResult`
  ("분석한 기준 커밋 명시")**. Bare-clone fetch is itself behind the concurrency cap to bound
  disk/network.
- Git operations sit behind a `GitRunner` interface; the **pure source-state transition
  logic** (given git facts → next state + baseline) is table-tested with a fake.

## 9. Security sandboxing (#4) — exact CLI invocations

> **Codex r1 (HIGH #5):** the two CLIs have **different** sandbox capabilities — the design
> must name exact, testable invocations, not "read-only flags." Verified against local
> `codex exec --help` and `claude --help`.

- **Fail-closed credentials (Codex r1, MEDIUM #6):** `secretbox.FromEnv` falls back to
  *plaintext* when `DS_APM_AI_CONFIG_ENCRYPTION_KEY` is unset (`secretbox.go:91`). CF-11 must
  **refuse to store or use a git credential for a private repo when encryption is unavailable**
  (return a config error; only public/credential-less repos are allowed in that mode). Never
  silently persist plaintext git creds.
- **Credential delivery:** decrypted **in-process only** at fetch; supplied to `git` via
  ephemeral `GIT_ASKPASS` + `GIT_TERMINAL_PROMPT=0` env, **never written to disk in plaintext,
  never logged** (logs redact).
- **Isolation:** per-run checkout `<base>/<org>/<repo>/<run_id>` at `baseline_commit`,
  **removed after the run** via deferred cleanup (even on failure/timeout/kill). The checkout
  is the **only writable surface** the agent has.
- **codex** (has an OS-level sandbox):
  `codex exec -s read-only -C <checkout> -m <model> --json` — `-s read-only` blocks
  model-generated shell writes at the OS layer; `-C` scopes the workspace to the checkout.
- **claude** (**no** OS read-only sandbox → application-level lockdown + cost cap):
  `claude -p <prompt> --append-system-prompt <sys> --model <m> --add-dir <checkout>
  --permission-mode default --allowed-tools "Read,Grep,Glob"
  --disallowed-tools "Bash,Write,Edit,WebFetch,WebSearch" --max-budget-usd <cap>`.
  Read-only is enforced by the tool allow/deny lists (NOT `--dangerously-skip-permissions`),
  the checkout being the only writable path, and **`--max-budget-usd` as a hard $ ceiling**.
  *Hardening option (not v1-blocking):* additionally wrap `claude` in an OS sandbox
  (bubblewrap/firejail, network limited to the model API). Noted as a follow-up.
- The agent receives **no secrets** in prompt/env. Model-API egress is required (the CLI calls
  the model); the agent's *tool* scope grants no other network/write capability.
- **Sandbox enforcement is tested** (Codex r1): T1 tests assert each CLI invocation is built
  with the exact read-only flags and that a write attempt in a fake-binary harness is rejected.
- **HITL:** `ProposedFix` is a suggestion delivered for human review; CF-11 **never applies
  changes** to any repo.

## 10. Trigger precision (#5)

`Trigger.Maybe` fires the gate chain (§6) only when the base predicate holds:

```
feature_on(org,service)
  ∧ cf1_unbound(signal)                    // binding.Status != Bound
  ∧ anomaly(signal)                        // AnomalySignal iface — FAIL-CLOSED (see below)
  ∧ severity(signal) >= min_severity       // default: high|critical
  ∧ service→repo resolvable
  ∧ admission succeeds (§6.2: dedup/cooldown ∧ budget ∧ queue-depth)
```

**Trigger gates admission, not concurrency (Codex r3).** The predicate decides whether to
*enqueue*; the global concurrency cap is a **worker-claim** condition (§6.3) applied later when
a worker picks up the queued run — it is deliberately **not** part of the trigger gate.

**Anomaly is fail-closed (Codex r1 #4; corrected Codex r2).** The hook receives **only labels
and annotations** (`dispatcher.go:658,663` → `hook.go:85`) — it does **not** carry the rule
type. `RuleTypeAnomaly` exists (`rule_type.go:12`) but lives in the *stored rule's* JSON
(`rule.go:13,19`), reachable only via a `ruleId`→stored-rule lookup (`ruleId` *is* on the
alert labels, `labels.go:26`). So for **v1 the `AnomalySignal` default requires an explicit
`anomaly` (or sustained) label/annotation on the alert** — set by the operator or by an
anomaly rule's annotations — and returns **false when absent**. The `ruleId`→rule-type lookup
(parse stored rule JSON, treat `RuleTypeAnomaly` as provenance) is a defined, **deferred
enrichment** behind the same `AnomalySignal` interface — not v1. Result: CF-11 does **not**
fire on every high-severity unbound alert. The legacy "unbound+severity eligible" behavior
survives only behind an off-by-default `allow_unbound_without_anomaly` flag with a loud
warning. Transient-error suppression = fail-closed anomaly + dedup + cooldown + min-severity
(+ future sustained-duration check, seam). The predicate is **pure and exhaustively
table-tested** (including the fail-closed default firing on zero alerts).

## 11. Seams (forbidden files — documented only, NOT edited in this WT)

| Seam | File | 1-line change (integration stage) |
|---|---|---|
| Trigger | `pkg/ruler/aigenerator/dispatchhook/hook.go` (unbound branch) | call `coderca.Trigger.Maybe(ctx, signal)` + inject Trigger dep |
| Route reg | `pkg/apiserver/signozapiserver/ruler.go` (`addRulerRoutes`) | `router.Handle(path, handler.New(provider.authZ.EditAccess(provider.codercaHandler.X), …)).Methods(…).GetError()` per method |
| Handler wiring | `pkg/ruler/signozruler/handler.go` + `pkg/signoz/handler.go` (`NewHandlers`) | construct coderca handler + add field/param |
| Server wiring | `cmd/community/server.go` | construct coderca engine/stores + inject into dispatcher + ruler factory |
| FE routing | FE router + `menuItems` | add settings page route |
| API codegen | `yarn generate:api` (orval) | after backend API merged |

CF-11 ships handlers/pages in **separate new files**; only the 1-line wirings above are left
as seams, listed here and in `SCOPE.md`.

## 12. Dependencies

- **Git access via the system `git` binary through `os/exec`** (mirrors `claudecli`/`codexcli`
  shelling out) → **no new Go module.** If `go-git` is preferred, that is a `go.mod` change =
  integration-stage coordination → **STOP-and-report**, not added unilaterally.
- **No new frontend deps** anticipated. Any addition → SCOPE + integration-stage per
  `TESTING.md §4`.

## 13. Build order — milestones (TDD; cost-control core is the gate)

Per `TESTING.md`: RED→GREEN→REFACTOR, test committed before impl. Pure cores first.
**Codex r1 (MEDIUM #8): the cost-control core (M1) must be PROVEN under a simulated flood
before any code path can spawn a CLI. M4 (FE) and the dispatch seam are not started until M1
holds.** This keeps the WT safely scoped and de-risks the #1 priority first.

- **M1 — Cost-control core (gating milestone, must pass before M3):**
  1. `codebase_config` + the cost tables (`coderca_run`, `coderca_admission`, `coderca_budget`,
     `coderca_skip_stat`) + migration `081` (Domain/Storable + secretbox; T0 store tests).
  2. **Atomic admission** `Admit()` — dedup unique index, per-day budget counter, queue depth
     (§6.2). T1: N concurrent admits for one key → exactly one run.
  3. **DB-backed claim/lease/heartbeat/reaper** + DB-counted global concurrency (§6.3). T1:
     concurrent claimers respect `max_concurrent_runs`; killed-lease run is reaped and re-runs
     ≤ `max_attempts`.
  4. Skip counters + sampled audit (§6.4).
  **Exit gate:** a flood-simulation T1 proves bounded runs/concurrency with **no double-admit**
  and **no duplicate DB finalization** across simulated restarts (duplicate CLI cost under a
  hard parent kill is bounded by `run_timeout` + `--max-budget-usd`, §6.5).
- **M2 — Pure analysis cores:** source-state transitions (`GitRunner` faked) + service→repo
  resolver + RCA-output parser (golden fixtures).
- **M3 — Adapters + orchestration:** `clirunner` (exact §9 flags, fake-binary integration incl.
  write-rejection + timeout) → `engine` wiring (admission→claim→source→cli→deliver→audit).
- **M4 — Surface (deferred until M1 proven):** HTTP handler (`coderca_handler.go`) + FE
  settings page. Seams (§11) documented only; **no shared-file edits in this WT.**

## 14. Acceptance criteria ↔ tests (1:1 per TESTING.md)

| Given / When / Then | Test |
|---|---|
| Given feature OFF, When unbound+anomaly, Then no run, no CLI | trigger gate table test (`feature_off`) |
| Given same dedup_key within cooldown, When re-fires, Then skip(`deduped`), prior result reused | dedup table test + runstore integration |
| Given org at `max_runs_per_day`, When fires, Then skip(`budget_exhausted`) | budget accounting table test |
| Given **N concurrent admits** for one dedup_key, When fired, Then **exactly one** run inserted | admission atomicity T1 (real SQLite/PG) |
| Given a `running` run whose lease expired (simulated crash), When reaper sweeps, Then re-queued and re-run ≤ `max_attempts` | lease/reaper T1 |
| Given N concurrent claimers and cap=1, When claiming, Then ≤ `max_concurrent_runs` runs go `running` (locked capacity row) | capacity-race T1 (real SQLite) |
| Given same error at cooldown−1m and cooldown+1m, When fired, Then 2nd admits (true sliding window, not bucket) | sliding-cooldown table test |
| Given a reaped run reclaimed by worker B, When original worker A finalizes, Then A's write is rejected (fencing token) | fencing T1 |
| Given encryption key unset, When saving a private-repo credential, Then config error (no plaintext stored) | fail-closed cred test |
| Given anomaly signal absent, When unbound high-sev alert fires, Then no run (fail-closed) | trigger gate table test (`no_anomaly`) |
| Given run exceeds `run_timeout`, When running, Then killed, status=`timeout` | clirunner integration (sleep binary) |
| Given a fetched repo, When triggered, Then baseline_commit pinned and echoed in result | source-state test + parser test |
| Given unmapped service, When fires, Then skip(`no_repo_mapping`) | resolver table test |
| Given valid config POST, When saved, Then secrets stored ciphertext, never returned plaintext | handler T1 + store test (mirrors ai_config) |

## 15. Open risks

- **CF-7 absence** — anomaly bridge is **fail-closed** (§10): until CF-7 lands, RCA fires only
  on alerts carrying anomaly-rule provenance / explicit anomaly label. Trade-off: fewer
  triggers, but no false-positive cost storm. Re-evaluate when CF-7 exposes a signal.
- **claude has no OS read-only sandbox** (§9) — enforced at application level (tool
  allow/deny + `--max-budget-usd` + disposable checkout). OS-sandbox wrap is a noted hardening
  follow-up; the checkout is the only writable surface regardless.
- **Disk pressure** from bare clones / per-run checkouts — bounded by concurrency cap +
  post-run cleanup; large monorepos may need shallow/partial clone (config flag, default
  shallow `--depth` fetch where history isn't needed — open question for §8).
- **Baseline drift** — analysis is pinned to a commit; if prod runs a different commit, the
  result echoes the analyzed baseline so humans can reconcile.
- **SQLite single-writer** — admission/claim transactions serialize under SQLite (less
  concurrency than PG `SKIP LOCKED`), but the single-winner invariant holds on both.

## 16. Codex review changelog (audit trail)

### Round 1 — 반영 (gpt-5.5, high reasoning; 1.18M in / 10.3k out)

| # | Sev | Finding | How addressed |
|---|---|---|---|
| 1 | CRIT | Cost gates non-atomic (TOCTOU across goroutines/replicas/restarts) | §6 rewritten: **atomic DB admission** in `RunInTxCtx` with unique `(org_id,dedup_key,window_bucket)` index + conditional budget/queue counters (§6.2) |
| 2 | CRIT | In-process worker pool unsafe across replicas/restarts | §6.3 **DB-backed lease** (`claimed_by`/`lease_until`/heartbeat/reaper) + DB-counted global concurrency; no leader election |
| 3 | HIGH | Per-rejection skip row + audit = write-amplification under flood | §6.4 **counters** (`hit_count`, per-(org,reason,day) upsert) + **sampled** audit |
| 4 | HIGH | Anomaly stub default fires RCA on every unbound alert | §10 **fail-closed**: keys off anomaly-rule provenance/label; legacy behavior behind off-by-default flag |
| 5 | HIGH | Sandboxing under-specified; CLIs differ | §9 **exact invocations** verified: codex `-s read-only -C`; claude tool allow/deny + `--max-budget-usd` (no OS sandbox); tested |
| 6 | MED | `secretbox.FromEnv` plaintext fallback | §9 **fail-closed** git creds when encryption unavailable |
| 7 | MED | Hook timeout is 5s in prod, not 1s | §5.1 corrected (`hook.go:26` default 1s, `signoz.go:420` prod 5s) |
| 8 | MED | Phase 0 scope too broad for one WT | §13 **milestones**; M1 cost-core must pass flood-sim before M3; FE/seam deferred |

Verdict r1: **DO NOT APPROVE** → all CRITICAL/HIGH + MEDIUMs addressed above; re-submitted round 2.

### Round 2 — 반영 (gpt-5.5, high reasoning; 1.76M in / 11.4k out)

r1 verdict on re-review: **#1,#3,#5,#6,#7,#8 RESOLVED; #2,#4 PARTIAL.** New flaws found in the
*fixes* and addressed:

| # | Sev | Finding | How addressed |
|---|---|---|---|
| 2 (cont.) / N1 | CRIT | Global concurrency still races: `count(running)<max` not lock-protected → 2 PG claimers both see 0 | §6.3 **locked `coderca_capacity` row** (conditional `UPDATE … WHERE running<max`) serializes the decision; SKIP LOCKED dropped |
| N2 | HIGH | Only SQLite registered (`provider.go:103`, modernc); no `FOR UPDATE SKIP LOCKED` | §6.2/6.3 **per-backend atomicity** spelled out: SQLite serialized writer (live), PG row-lock/SERIALIZABLE (future); SKIP LOCKED removed |
| 4 (cont.) / N3 | HIGH | Anomaly provenance not at the hook; `ruleType` needs ruleId→rule lookup | §10 **v1 = explicit anomaly label/annotation**; ruleId→rule-type lookup deferred behind same iface |
| N4 | MED | Bucketed `floor(now/window)` admits twice across bucket edge | §6.2 **sliding `last_admitted_at`**, key `(org_id, dedup_key)` |
| N5 | MED | Reaper can double-run a paused-but-live worker (heartbeat lag) | §6.3 **fencing token** + **`lease_ttl ≥ run_timeout+grace`**; "no double-run" claim made precise |

Re-submitted round 3.

### Round 3 — 반영 (gpt-5.5, high reasoning; 504K in / 10K out)

r3: **#4, N1, N2, N3, N4 RESOLVED; #2, N5 PARTIAL.** Mechanism attack found **no lock-order
deadlock** and confirmed sliding-window dedup safe (no ABA). 3 remaining blockers fixed:

| # | Sev | Finding | How addressed |
|---|---|---|---|
| R3-1 | HIGH | SIGKILLed parent orphans the child CLI; lease expiry starts a 2nd CLI (fencing protects DB, not CLI cost) | §6.5 **process-group + `Pdeathsig` + startup orphan sweep**; §6.3 claim **honestly downgraded** to "no duplicate DB finalization" + bounded CLI-cost risk |
| R3-2 | MED | Capacity decrement could undercount via stale finalizer | §6.3 **fenced, affected-row-checked, same-tx decrement contract** (zero-row stale update must not decrement) |
| R3-3 | MED | Trigger predicate wrongly included claim-time concurrency | §10 predicate trimmed to admission only; concurrency noted as worker-claim condition |

### Round 4 — **APPROVE** (gpt-5.5, high reasoning; 631K in / 7.5K out)

R3-1, R3-2, R3-3 all **RESOLVED**; **no new blocking flaws**. Codex verified the subprocess
semantics against the Go toolchain source (`Setpgid`/`Pdeathsig` via `PR_SET_PDEATHSIG`) and
the DB assumptions against the repo (`RunInTxCtx` `bun.go:31`, no serializable default
`bun.go:38`, SQLite/modernc the only registered store `provider.go:103`). Two non-blocking
nits folded in: M1 exit-gate wording (§13) and `Pdeathsig`/orphan-sweep caveats (§6.5).

> **Loop result:** 4 rounds, ~37 min wall-clock, ≈4.07M input / ≈39K output tokens. Arc:
> r1 reframed the hard problem as *atomic, distributed* cost control (not the AI call); r2/r3
> made the DB admission/lease/concurrency/fencing correct and honest; r4 clean **APPROVE**.
