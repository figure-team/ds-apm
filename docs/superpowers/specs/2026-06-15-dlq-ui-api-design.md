# DLQ UI/API 설계 — 전송 실패 내역 조회 및 재전송

**날짜**: 2026-06-15  
**상태**: approved  
**관련 스펙**: CF-5 (무유실·멱등 재처리), Story 5.1, 5.2  

---

## 1. 목표

운영자가 알림 전송 실패 내역(DLQ)을 UI에서 조회하고, 선택한 항목을 멱등하게 재전송할 수 있도록 한다. 백엔드는 이미 구현된 `JSONLDeadLetterSink`, `ReplayLedger`, `ReadEntries`, `Sign/Verify`를 활용하며, 신규 `DLQManager`가 이를 조율한다.

---

## 2. 범위

**In scope:**
- DLQ 항목 조회 API (채널·상태 필터)
- 단건 / 일괄 재전송 API (멱등 보장)
- Notification Channels 페이지에 탭으로 UI 추가
- 재전송 상태 3종 표시: 대기중 / 재전송됨 / 재전송 실패

**Out of scope:**
- HMAC 서명 검증 (CF-5 Story 5.3, 정책 미정)
- DLQ 항목 수동 삭제
- 재전송 스케줄링 / 자동 재시도

---

## 3. 백엔드 아키텍처

### 3.1 DLQManager

**파일**: `pkg/alertmanager/alertmanagerserver/dlqmanager.go`

```go
type DLQManager struct {
    dlqPath  string
    ledger   *dlq.ReplayLedger
    sidecar  *FailureSidecar       // {event_id} JSON Lines 파일
    // notifyFn은 채널 이름과 alerts를 받아 해당 채널의 notifier를 직접 호출한다.
    // dispatcher의 라우팅을 거치지 않고 notificationManager.GetNotifier(channel).Notify() 를 호출.
    notifyFn func(ctx context.Context, channel string, alerts []*types.Alert) error
}
```

**FailureSidecar** (`dlqmanager_sidecar.go`):
- 파일 경로: `<dlqPath>.replay-failures` (예: `var/ds-apm/alert-dlq.jsonl.replay-failures`)
- append-only, 한 줄에 `event_id` 하나
- `Record(eventID string)` — 실패 기록
- `Has(eventID string) bool` — 존재 여부 확인
- 재시작 시 파일 스캔으로 in-memory set 재구성 (ReplayLedger 동일 패턴)

**DLQManager 연결 (`server.go`)**:
- `Server` 구조체에 `dlqManager *DLQManager` 필드 추가
- `New()` 내부에서 `dlqSink`와 동일하게 `SIGNOZ_DLQ_PATH` 환경변수로 초기화
- `notifyFn`은 `server.notificationManager`의 채널별 notifier를 클로저로 주입
- `signozalertmanager/provider.go`의 `ListDLQEntries` / `ReplayDLQEntries`는 `Server.dlqManager`를 위임 호출

### 3.2 Status 판정 로직

```
ledger.Has(eventID)  → "replayed"
sidecar.Has(eventID) → "replay_failed"
otherwise            → "pending"
```

### 3.3 DLQEntry DTO

```go
// pkg/alertmanager/alertmanagerserver/dlqmanager.go
type DLQEntry struct {
    EventID  string    `json:"event_id"`
    Channel  string    `json:"channel"`
    Payload  []byte    `json:"payload"`  // base64 인코딩 (API 응답 시)
    FailedAt time.Time `json:"failed_at"`
    Reason   string    `json:"reason"`
    Status   string    `json:"status"`   // "pending" | "replayed" | "replay_failed"
}
```

### 3.4 Replay 실행 흐름

```
ReplayDLQEntries(ctx, orgID, eventIDs):
  entries := ReadEntries(dlqPath)
  for each eventID in eventIDs:
    entry := find entry by eventID
    // round=0 고정: 초기 재전송. 추후 재시도 횟수 추적이 필요하면 round를 증가시키는 방식으로 확장.
    if !ledger.MarkIfNew(IdempotencyKey(entry.EventID, entry.Channel, 0)):
      → skip (already replayed)
    alerts := json.Unmarshal(entry.Payload)
    err := notifyFn(ctx, entry.Channel, alerts)
    if err != nil:
      → sidecar.Record(entry.EventID)
      → count failed++
    else:
      → count replayed++
  return {replayed, skipped, failed}
```

### 3.5 Alertmanager 인터페이스 확장

```go
// pkg/alertmanager/alertmanager.go
ListDLQEntries(ctx context.Context, orgID, channel, status string) ([]*DLQEntry, error)
ReplayDLQEntries(ctx context.Context, orgID string, eventIDs []string) (*ReplayResult, error)

type ReplayResult struct {
    Replayed int `json:"replayed"`
    Skipped  int `json:"skipped"`
    Failed   int `json:"failed"`
}
```

---

## 4. REST API

### GET /api/v1/alertmanager/dlq/entries

**쿼리 파라미터**:
| 파라미터 | 타입   | 설명                                      |
|--------|--------|-------------------------------------------|
| channel | string | 선택. 채널명 필터 (예: `slack`)           |
| status  | string | 선택. `pending` \| `replayed` \| `replay_failed` |

**응답 200**:
```json
{
  "data": [
    {
      "event_id": "abc123def456",
      "channel": "slack",
      "failed_at": "2026-06-15T09:23:11Z",
      "reason": "connection refused: slack.com:443",
      "status": "pending",
      "payload": "<base64-encoded-json>"
    }
  ]
}
```

### POST /api/v1/alertmanager/dlq/replay

**요청 바디**:
```json
{ "event_ids": ["abc123def456", "xyz789abc123"] }
```

**응답 200**:
```json
{
  "data": {
    "replayed": 1,
    "skipped": 1,
    "failed": 0
  }
}
```

**멱등성**: 동일 `event_id`를 두 번 요청해도 Ledger에 의해 skip 처리. `skipped` 카운트에 포함.

### 라우트 등록

`pkg/query-service/app/http_handler.go` `RegisterRoutes()` 내:
```go
r.HandleFunc("/api/v1/alertmanager/dlq/entries", am.GetDLQEntries).Methods("GET")
r.HandleFunc("/api/v1/alertmanager/dlq/replay",  am.ReplayDLQEntries).Methods("POST")
```

핸들러 위치: `pkg/alertmanager/signozalertmanager/handler.go` (기존 패턴 동일)

---

## 5. 프론트엔드

### 5.1 위치

`frontend/src/container/AllAlertChannels/index.tsx`에 Ant Design `<Tabs>` 추가:
- **Tab 1**: 채널 목록 (기존 `AlertChannels` 컴포넌트)
- **Tab 2**: 전송 실패 내역 (신규 `DLQFailures` 컨테이너)

### 5.2 컴포넌트 트리

```
AllAlertChannels/index.tsx
└── <Tabs>
    ├── Tab "채널 목록"       → AlertChannels (기존, 변경 없음)
    └── Tab "전송 실패 내역"  → frontend/src/container/DLQFailures/
        ├── index.tsx            (데이터 fetch, 상태 관리)
        ├── DLQFilters.tsx       (채널 · 상태 Select 드롭다운)
        ├── DLQBulkActionBar.tsx (선택 시 출현하는 플로팅 바)
        ├── DLQTable.tsx         (ResizeTable 기반)
        └── DLQPayloadDrawer.tsx (Ant Design Drawer, payload JSON 표시)

frontend/src/api/dlq/
    ├── getDLQEntries.ts   (GET /api/v1/alertmanager/dlq/entries)
    └── replayDLQEntries.ts (POST /api/v1/alertmanager/dlq/replay)
```

API 클라이언트는 `frontend/src/api/channels/getAll.ts` 패턴을 따른다.

### 5.3 테이블 컬럼

| 컬럼        | 설명                                          |
|------------|-----------------------------------------------|
| ☐ 체크박스  | 헤더 클릭 시 전체 선택                         |
| Alert ID   | `event_id` (monospace, 12자 truncate + tooltip) |
| 채널        | Tag 배지                                       |
| 실패 이유   | `reason` 텍스트                                |
| 실패 시각   | `failed_at` 로컬 시간                          |
| 상태        | 배지: 대기중(초록) / 재전송됨(파랑) / 재전송 실패(빨강) |
| 페이로드    | "보기" 버튼 → DLQPayloadDrawer 열림            |

### 5.4 플로팅 액션 바

- 1개 이상 선택 시 테이블 상단에 출현
- `"N개 선택됨 · [↩ 재전송] [선택 해제]"`
- 재전송 클릭 → `POST /dlq/replay` → 완료 후 entries refetch → 상태 자동 갱신

### 5.5 필터

- **채널 드롭다운**: API에서 entries를 받아 채널 목록 동적 생성 (전체 / slack / webhook / …)
- **상태 드롭다운**: 전체 / 대기중 / 재전송됨 / 재전송 실패

### 5.6 페이로드 Drawer

- Ant Design `<Drawer>` (우측 슬라이드)
- base64 디코딩 후 JSON pretty-print (`JSON.stringify(JSON.parse(decoded), null, 2)`)
- 복사 버튼 제공

---

## 6. 테스트 전략

### 백엔드

| 파일 | 테스트 대상 |
|------|-----------|
| `dlqmanager_test.go` | ListEntries 상태 병합, Replay 성공/실패/skip, FailureSidecar 영속성 |
| `handler_dlq_test.go` | GET 채널·상태 필터, POST replay 응답 카운트 |

기존 `dispatcher_dlq_test.go` 패턴 (fakeDLQSink) 동일하게 적용.

### 프론트엔드

기존 `AllAlertChannels/__tests__/` 패턴 따름:
- 탭 렌더링 및 전환 테스트
- 체크박스 선택 → 액션 바 출현 테스트
- API mock으로 상태별 배지 렌더링 검증
- 재전송 후 refetch 검증

---

## 7. 파일 변경 요약

| 파일 | 변경 종류 |
|------|---------|
| `pkg/alertmanager/alertmanager.go` | `ListDLQEntries`, `ReplayDLQEntries` 인터페이스 추가 |
| `pkg/alertmanager/alertmanagerserver/dlqmanager.go` | 신규 — DLQManager, FailureSidecar, DLQEntry |
| `pkg/alertmanager/alertmanagerserver/dlqmanager_test.go` | 신규 — 단위 테스트 |
| `pkg/alertmanager/signozalertmanager/handler.go` | `GetDLQEntries`, `ReplayDLQEntries` 핸들러 추가 |
| `pkg/alertmanager/signozalertmanager/provider.go` | 인터페이스 구현 추가 |
| `pkg/query-service/app/http_handler.go` | 라우트 2개 등록 |
| `frontend/src/container/AllAlertChannels/index.tsx` | Tabs 래핑 |
| `frontend/src/container/DLQFailures/` | 신규 컨테이너 (5개 파일) |
| `frontend/src/api/dlq/` | 신규 API 클라이언트 |
