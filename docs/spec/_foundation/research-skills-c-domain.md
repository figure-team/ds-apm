# 01-C. Observability/SRE 도메인 문서 패턴 리서치

> Agent C 결과물 (분할). 이후 `research-skills.md`로 통합 예정.
> 작성일: 2026-05-28
> 대상: DS-APM (SigNoz observability + AI runbook handoff layer)
> 범위: Use Case 명세, 기능명세서 작성에 필요한 도메인 표준 패턴

---

## 0. Executive Summary — DS-APM에 적용할 5가지 핵심 결론

| # | 결론 | 적용 위치 |
|---|---|---|
| 1 | **Use Case는 swimlane sequence diagram 1장 + state machine 1장 조합**이 업계 표준. Alert → SOP → Draft → Approval → Dispatch는 actor가 4~5개(Monitoring/AI Engine/Operator/Dispatcher/Channel)이므로 swimlane sequence가 필연 | Use Case 문서 |
| 2 | **runbook 1건은 13개 필드 표준 구조**: Title, Owner, Severity, Trigger, Pre-checks, Steps, Verification, Rollback, Escalation, SLA, Related Alerts, Last Updated, Postmortem Links | 기능명세서 §SOP Schema |
| 3 | **알람 메타데이터는 Prometheus Alertmanager + PagerDuty Events API v2의 합집합**을 따르라. 핵심 필드: `alertname`, `severity`, `status`, `startsAt`, `endsAt`, `labels`, `annotations.runbook_url`, `annotations.summary`, `annotations.description`, `generatorURL`, `fingerprint`, `dedup_key` | 기능명세서 §Alert Schema |
| 4 | **PII redaction은 OpenTelemetry Collector 패턴**(Attribute / Filter / Redaction / Transform Processor)을 명세에 박을 것. "어디서 redaction하나"의 답은 항상 **as early as possible, ideally at instrumentation** | 기능명세서 §PII Policy |
| 5 | **DLQ idempotency key는 event_id 기반 deterministic key**가 표준. TTL은 retry window를 초과해야 함. `(alert_fingerprint, channel_id, dispatch_attempt_round)` 튜플 권장 | 기능명세서 §DLQ |

---

## 1. Google SRE Book / Workbook — 핵심 권장 사항

### 1.1 Practical Alerting (SRE Book Ch.6)

- **알람은 high-level service objective에 걸어라**. 컴포넌트 단위가 아니라 사용자 가시 증상(latency, error rate) 위에서 알람을 정의한다.
- White-box monitoring + black-box prober 둘 다 필요. "you aren't aware of what the users see. You only see the queries that arrive at the target."
- 좋은 알람 4원칙:
  1. **Minimum duration** (flapping 방지) → DS-APM에서는 `for: 5m` 같은 hold 필드 필수
  2. **Contextual information** (어느 컴포넌트, 어떤 값에 trigger됐는지)
  3. **Intelligent routing** (Alertmanager 같은 dedup/inhibit 계층)
  4. **Severity-based routing** — page-worthy는 on-call, subcritical은 ticket queue로
- Mantra: *"May the queries flow, and the pager stay silent."*

### 1.2 Managing Incidents (SRE Book Ch.14) + IMAG (Workbook)

**Incident Command System(ICS) 기반 4개 역할:**

| 역할 | 책임 | DS-APM 매핑 |
|---|---|---|
| Incident Commander (IC) | "single source of truth"; 우선순위 결정, 역할 위임 | 운영자 (Operator) |
| Operations Lead (Ops) | 시스템에 실제 손대는 유일한 그룹 | 운영자 (Operator), runbook executor |
| Communications Lead | stakeholder에게 주기적 업데이트 | DS-APM dispatch layer (Slack/MSTeams broadcast) |
| Planning Lead | bug 등록, 식사 주문, handoff, 시스템 정상상태와의 격차 추적 | postmortem 후속 ticket |

**Declaration criteria** (셋 중 하나라도 yes면 incident 선언):
- "Do you need to involve a second team in fixing the problem?"
- "Is the outage visible to customers?"
- "Is the issue unsolved even after an hour's concentrated analysis?"

**Handoff ceremony** (DS-APM 운영자 교대에 반드시 모방):
> "You're now the incident commander, okay?" — 명시적 인수 + firm acknowledgment 수신 전까지 콜에서 나가지 않는다.

**Three Cs**: Coordinate, Communicate, Control.

**Best practices**:
- Declare early
- 중앙 communication hub (war room)
- **Mitigation first, root-cause later**
- Working document로 디버깅 기록 보존 (DS-APM의 incident timeline)
- 4시간마다 on-call rotate

### 1.3 Effective Troubleshooting (SRE Book Ch.12)

Hypothetico-deductive 6단계: **Problem Report → Triage → Examine → Diagnose → Test/Treat → Cure**.

> "Your first response in a major outage may be to start troubleshooting...Ignore that instinct!" — 일단 mitigate.

**Runbook을 위한 시스템 요구사항**:
- 동적 verbosity 레벨 (재시작 없이)
- 구조화된 로그 포맷
- 컴포넌트 간 observable interface
- Request tracing (Dapper-like)
- 최근 RPC, error rate, latency histogram을 노출하는 state endpoint
- 배포/config 변경 이력과 성능의 correlation

**Pitfalls** (DS-APM의 AI draft가 절대 하면 안 되는 것):
- 과거 패턴을 잘못 적용 ("since it happened once...")
- Correlation을 causation으로 착각
- Occam's razor 무시 — 단순한 설명을 지나치고 복잡한 이론을 쫓음

### 1.4 Postmortem Culture (SRE Book Ch.15)

**Trigger 기준** (DS-APM은 SEV-1/SEV-2 자동 발화):
- User-visible downtime/degradation > 임계치
- 모든 data loss
- 엔지니어가 수동 개입 (rollback, traffic reroute)
- Resolution time > limit
- 모니터링이 놓쳐서 수동 발견

**Blameless 원칙**: "focuses on identifying the contributing causes of the incident without indicting any individual or team for bad or inappropriate behavior."

---

## 2. PagerDuty Incident Response Documentation

출처: https://response.pagerduty.com/

### 2.1 4-phase 구조

1. **Before an Incident** — 정의, severity, 역할, communication 프로토콜
2. **During an Incident** — 실시간 대응
3. **After an Incident** — postmortem, 학습
4. **Crisis Response** — 기술 incident를 넘는 조직 차원 위기

### 2.2 Severity Levels (DS-APM의 알람 severity 매핑 기준)

| Level | 정의 (PagerDuty quote) | 응답 |
|---|---|---|
| **SEV-1** | "Critical issue that warrants public notification and liaison with executive teams" | IC paging + major incident procedure |
| **SEV-2** | "Critical system issue actively impacting many customers' ability to use the product" | IC paging + major incident |
| **SEV-3** | "Stability or minor customer-impacting issues that require immediate attention from service owners" | High-urgency page to service team |
| **SEV-4** | "Minor issues requiring action, but not affecting customer ability to use the product" | Low-urgency page, top-priority work |
| **SEV-5** | "Cosmetic issues or bugs, not affecting customer ability to use the product" | JIRA ticket only |

**핵심 룰**: SEV-2 이상은 모두 **major incident**. 불확실하면 **"Always Assume The Worst"** → 더 높은 severity로 처리.

### 2.3 Incident Roles (6개)

출처: https://response.pagerduty.com/before/different_roles/

| 역할 | 책임 |
|---|---|
| **Incident Commander (IC)** | "single source of truth of what is currently happening"; channel 준비, drive to resolution, delegation, 외부 communication 관리, postmortem 감독 |
| **Deputy** | IC의 직접 지원. "hot standby" Incident Commander. Timer 관찰, missed item callback, 콜 관리, 참가자 제거 권한 |
| **Scribe** | 타임라인 documentation, 콜 녹음, Slack에 액션 노트, status report 기록 |
| **Subject Matter Expert (SME)** | 도메인 전문가. CAN report (Conditions / Actions / Needs) 형식으로 보고 |
| **Customer Liaison** | 외부 communication 초안, 영향받은 고객 통지, 승인된 메시지 public 채널 게시 |
| **Internal Liaison** | 내부 리소스 동원 (Finance, Legal, Marketing 등 호출), 내부 stakeholder에 status 제공 |

> **참고**: 작은 incident에서는 한 사람이 여러 역할 겸직 가능. Deputy가 Scribe + Internal Liaison 겸하는 식.

### 2.4 During an Incident — 표준 흐름

1. **Initial Response**: 콜과 채팅 join. IC 없으면 `!ic page`로 호출
2. **IC Takes Command**: 역할 선언, Deputy/Scribe 지정, 원인 가설, SME에 조사 위임
3. **Investigation & Repair**: SME가 액션 제안 (rollback, restart, throttle...), IC가 결정
4. **Recovery & Closeout**: IC가 resolve 선언, 콜 종료, 남은 토의는 Slack으로

**Communication cadence**: Internal Liaison이 약 **30분마다** executive team에 Slack status 업데이트.

### 2.5 Postmortem Template Fields (전체 인용)

출처: https://response.pagerduty.com/after/post_mortem_template/

- **Postmortem Owner** — 담당자
- **Meeting Scheduled For** — SEV-1은 3 calendar days, SEV-2는 5 business days 내
- **Call Recording** — 콜 녹음 링크
- **Overview** — 1~2 문장 ("contributing factors, timeline summary, impact")
- **What Happened** — 짧은 서술
- **Contributing Factors** — 기여 조건 + "exacerbated the issue" 액션
- **Resolution** — 임시 fix + 장기 해결책
- **Impact** — 정량 지표
  - Time in SEV-1 (분)
  - Time in SEV-2 (분)
  - Notifications Delivered out of SLA (% + count)
  - Events Dropped / Not Accepted (% + count)
  - Accounts Affected (count)
  - Users Affected (count)
  - Support Requests Raised (ticket 링크)
- **Responders** — IC, Scribe 등 명단
- **Timeline** — UTC 타임스탬프 + 설명 + 도구/로그 링크
- **How'd We Do?** — What Went Well / What Didn't Go So Well
- **Action Items** — JIRA ticket (sev1_YYYYMMDD, sev1 라벨)
- **Messaging** — Internal Email + External Message

**Postmortem 상태 머신**: Draft → In Review → Reviewed → Closed.

---

## 3. GitLab Handbook — Runbook 구조

출처: https://runbooks.gitlab.com/, https://docs.gitlab.com/user/project/clusters/runbooks/

### 3.1 조직 원칙

- **Service-based 디렉터리 구조**: 서비스 카탈로그의 각 서비스가 자체 폴더 + auto-generated README
- Top-level은 공식 서비스 이름만
- 정체불명 문서는 "Uncategorized" 폴더

### 3.2 README.md 권장 섹션 (Service 단위)

- Summary
- Architecture
- Performance
- Scalability
- Availability
- Durability
- Security/Compliance
- Monitoring/Alerting
- Further documentation 링크

### 3.3 Individual Runbook 권장 구조 (DS-APM SOP 스키마와 직결)

**Troubleshooting Section** (symptom별로 정리):
- Issue 설명
- 식별 가능한 indicator (alerts, metrics, logs)
- Step-by-step 해결책

**Maintenance Section**:
- 일반 운영 작업 (failover, backup)

**Key principle**: "as short and concise as possible" + "complete enough to be executed without further research". 배경 정보는 README로 링크.

### 3.4 Static vs. Executable Runbook

- **Static**: decision tree 또는 step-by-step 매뉴얼
- **Executable** (GitLab의 권장 방향): JupyterHub + Nurtch Rubix로 pre-written code block 실행 가능 → DS-APM의 AI draft 단계가 여기에 해당

---

## 4. OpenTelemetry / SigNoz 문서 구조 및 표준 필드

### 4.1 OpenTelemetry Resource Semantic Attributes (알람 페이로드 표준 필드)

출처: https://opentelemetry.io/docs/specs/semconv/resource/

**Core service**:
- `service.name` — 논리적 그룹 식별자
- `service.namespace` — 조직 단위
- `service.version` — semantic / git hash / 커스텀

**Deployment**:
- `deployment.environment` — prod/staging/dev
- `telemetry.sdk.{language|name|version}`

**Infrastructure**:
- `host.name`
- `k8s.*` (cluster, namespace, pod, node)
- `container.*`
- `process.*`
- `cloud.provider` (aws/gcp/azure/...)

**Context**:
- `os.*`, `device.*`, `browser.*`
- `telemetry.distro.name`

→ DS-APM 알람 payload는 **반드시 이 resource attribute 전체를 그대로 캐리**해야 한다. SigNoz가 OTel-native이므로 자연스럽게 들어옴.

### 4.2 SigNoz Alerts Management

출처: https://signoz.io/docs/alerts/, https://signoz.io/docs/userguide/alerts-management/

**지원 alert 타입 5종**:
1. Metric-based
2. Log-based
3. Trace-based
4. Exceptions-based
5. Anomaly-based

**Alert 메타데이터 필드**:
- Status (enabled/disabled, firing/resolved)
- Alert Name
- Severity
- Labels
- Firing Since (timestamp)

**Notification 채널**: Slack, PagerDuty, Opsgenie, **MS Teams**, Email, Webhook.

**Configuration tab**:
- Routing Policy — labels/severity 기반
- Planned Maintenance — silencing window

**참고**: SigNoz는 Terraform 지원으로 alert as code 가능 → DS-APM도 SOP를 GitOps 패턴으로 관리하는 게 자연스러움.

### 4.3 OpenTelemetry 공식 문서 구조 (DS-APM 문서 구조 벤치마크)

OTel 공식 사이트는 다음 5단으로 분리:
1. **Concepts** — 시그널 (traces/metrics/logs/baggage), 의미론
2. **Instrumentation** — 언어별 가이드
3. **Specs / Semantic Conventions** — 표준 attribute, schema
4. **Collector** — 운영자용 (processor, exporter)
5. **Security** — Handling Sensitive Data 같은 cross-cutting concern

→ DS-APM 문서도 `Concepts / Usage / API Reference / Operations / Security & PII` 5분할 권장.

---

## 5. Prometheus Alertmanager — Webhook Payload Schema (DS-APM ingress 표준)

### 5.1 Alert object 필드

| Field | Type | 설명 |
|---|---|---|
| `Status` | string | `firing` 또는 `resolved` |
| `Labels` | map | "A set of labels to be attached to the alert" |
| `Annotations` | map | "A set of annotations for the alert" |
| `StartsAt` | time | 알람 시작 시각 |
| `EndsAt` | time | 종료 시각 (알 수 있을 때만) |
| `GeneratorURL` | string | "A backlink which identifies the causing entity of this alert" |
| `Fingerprint` | string | "Fingerprint that can be used to identify the alert" — **idempotency key의 시드** |

### 5.2 Webhook 페이로드 전체 예시 (DS-APM ingress가 그대로 받아야 하는 포맷)

```json
{
  "receiver": "ds-apm-webhook",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "PaymentService5xxHigh",
        "severity": "critical",
        "service": "payment-api",
        "deployment_environment": "production",
        "team": "payments"
      },
      "annotations": {
        "summary": "Payment API 5xx rate above 5% for 5m",
        "description": "Error rate is 12.4% on payment-api over the last 5 minutes",
        "runbook_url": "https://wiki.example.com/runbooks/payment-5xx",
        "dashboard_url": "https://signoz.example.com/dashboard/payment"
      },
      "startsAt": "2026-05-28T14:23:11.000Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "https://signoz.example.com/alerts/123"
    }
  ],
  "groupLabels": { "alertname": "PaymentService5xxHigh" },
  "commonLabels": {
    "alertname": "PaymentService5xxHigh",
    "severity": "critical",
    "service": "payment-api"
  },
  "commonAnnotations": {
    "summary": "Payment API 5xx rate above 5% for 5m"
  },
  "externalURL": "https://signoz.example.com",
  "version": "4",
  "groupKey": "{}:{alertname=\"PaymentService5xxHigh\"}"
}
```

**관례**:
- `severity` label은 `info` | `warning` | `critical` (Prometheus 커뮤니티 표준)
- `annotations.runbook_url` — DS-APM이 SOP grounding할 때 1차 키
- `annotations.summary`, `annotations.description` — 본문 그라운딩 컨텍스트
- `fingerprint` — Alertmanager가 자동 생성. DS-APM의 idempotency key에 그대로 활용 가능

---

## 6. PagerDuty Events API v2 — Channel Egress 표준

출처: https://developer.pagerduty.com/docs/events-api-v2/

```json
{
  "routing_key": "YOUR_INTEGRATION_KEY",
  "event_action": "trigger",
  "dedup_key": "payment-api/PaymentService5xxHigh/2026-05-28T14:23:11Z",
  "payload": {
    "summary": "Payment API 5xx rate above 5% for 5m",
    "timestamp": "2026-05-28T14:23:11Z",
    "source": "signoz.production.payment-api",
    "severity": "critical",
    "component": "payment-api",
    "group": "payments",
    "class": "5xx_error_rate",
    "custom_details": {
      "error_rate": "12.4%",
      "runbook_url": "https://wiki.example.com/runbooks/payment-5xx",
      "ds_apm_draft_id": "draft_01HXYZ...",
      "ds_apm_approval_status": "approved"
    }
  }
}
```

**핵심 필드**:
- `dedup_key` — 같은 키로 재전송 시 기존 alert에 병합. **DS-APM의 idempotency 키와 1:1 매핑**.
- `severity` — `critical | error | warning | info` (Prometheus와 매핑 표 필요)
- `source` — FQDN 권장
- `custom_details` — 자유 schema. DS-APM은 여기에 draft_id, approver, runbook_url 박는다.

---

## 7. Slack & MS Teams Channel Payload 표준

### 7.1 Slack Incoming Webhook (Block Kit)

```json
{
  "blocks": [
    { "type": "header", "text": { "type": "plain_text", "text": "[SEV-2] Payment API 5xx" } },
    { "type": "section", "text": { "type": "mrkdwn", "text": "*Service:* payment-api\n*Env:* production\n*Error rate:* 12.4%" } },
    { "type": "section", "fields": [
        { "type": "mrkdwn", "text": "*Started:*\n2026-05-28 14:23 UTC" },
        { "type": "mrkdwn", "text": "*Owner:*\npayments@example.com" }
    ]},
    { "type": "divider" },
    { "type": "section", "text": { "type": "mrkdwn", "text": "*Runbook draft (AI, approved):*\n1. Check payment-gateway upstream\n2. Verify recent deploy SHA `abc123`\n3. If gateway healthy → rollback" }},
    { "type": "actions", "elements": [
        { "type": "button", "text": { "type": "plain_text", "text": "Open in SigNoz" }, "url": "https://signoz.example.com/alerts/123" },
        { "type": "button", "text": { "type": "plain_text", "text": "Full Runbook" }, "url": "https://wiki.example.com/runbooks/payment-5xx" }
    ]}
  ]
}
```

**제약**: incoming webhook은 default channel/username/icon override 불가.

### 7.2 MS Teams Adaptive Card

```json
{
  "type": "message",
  "attachments": [{
    "contentType": "application/vnd.microsoft.card.adaptive",
    "content": {
      "$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
      "type": "AdaptiveCard",
      "version": "1.4",
      "body": [
        { "type": "TextBlock", "size": "Large", "weight": "Bolder", "text": "[SEV-2] Payment API 5xx" },
        { "type": "FactSet", "facts": [
          { "title": "Service:", "value": "payment-api" },
          { "title": "Env:", "value": "production" },
          { "title": "Error rate:", "value": "12.4%" },
          { "title": "Started:", "value": "2026-05-28 14:23 UTC" }
        ]},
        { "type": "TextBlock", "wrap": true, "text": "**Runbook draft (AI, approved)**: 1. Check payment-gateway upstream  2. Verify recent deploy SHA `abc123`  3. If gateway healthy → rollback" }
      ],
      "actions": [
        { "type": "Action.OpenUrl", "title": "Open in SigNoz", "url": "https://signoz.example.com/alerts/123" },
        { "type": "Action.OpenUrl", "title": "Full Runbook", "url": "https://wiki.example.com/runbooks/payment-5xx" }
      ]
    }
  }]
}
```

**중요 제약**: Teams incoming webhook은 `Action.OpenUrl`, `Action.ShowCard`, `Action.ToggleVisibility`만 지원. **`Action.Submit`은 작동하지 않음** — DS-APM에서 Teams로 "approve/reject" 같은 양방향 버튼은 못 박는다.

---

## 8. ITIL Incident Management — Priority Matrix

| Impact \ Urgency | High | Medium | Low |
|---|---|---|---|
| **Catastrophic** | P1 | P1 | P2 |
| **Major** | P1 | P2 | P3 |
| **Minor** | P2 | P3 | P4 |
| **Trivial** | P3 | P4 | P4 |

**ITIL 정의**:
- **Impact**: 비즈니스 disruption의 크기 (얼마나 광범위한가)
- **Urgency**: 해결 시급성 (얼마나 빨리 fix 필요한가)
- **Priority = Impact × Urgency**

→ DS-APM은 SigNoz/Prometheus의 `severity` 외에 **`urgency`와 `impact` 필드를 분리해서 받는 것**이 도메인 정석. SEV-2지만 urgency=low (점진적 영향, 야간에 fix 가능)인 케이스가 실재함.

---

## 9. PII Redaction Policy (OpenTelemetry 가이드 기반)

출처: https://opentelemetry.io/docs/security/handling-sensitive-data/

### 9.1 원칙

1. **Data minimization** — 관찰 목적에 필요한 데이터만 수집
2. **As early as possible** — instrumentation layer에서 거르는 게 최선, Collector에서 거르는 게 차선
3. **Hash는 안전하지 않을 수 있음** — "hashes are reversible in practice if the input space is small and predictable"

### 9.2 Collector Processor 4종 (DS-APM 명세에 박을 표준)

| Processor | 용도 |
|---|---|
| **Attributes Processor** | 특정 attribute remove/modify (예: `user.email` 삭제) |
| **Filter Processor** | sensitive data가 들어간 span/metric 전체를 차단 |
| **Redaction Processor** | allowlist 외 attribute 삭제 |
| **Transform Processor** | regex 기반 — credit card, IP truncation, ID hashing 등 |

### 9.3 Monitoring the redactor

> "Send incidents to security/on-call if collectors drop or redact more than a threshold number of fields per minute" — 갑작스러운 redaction spike는 **새 코드 경로에서 PII가 새고 있다는 시그널**.

→ DS-APM은 **redaction rate metric** 자체를 alert source로 끌어와야 한다.

---

## 10. DLQ + Idempotent Replay 설계 표준

### 10.1 Idempotency Key 설계 규칙

출처: https://hookdeck.com/webhooks/guides/implement-webhook-idempotency

1. **Deterministic derivation** — 같은 event는 항상 같은 key 생성. 랜덤 UUID 금지.
2. **Event-inherent ID 활용** — 페이로드 안의 자연 키 (Alertmanager의 `fingerprint`, PagerDuty의 `dedup_key`) 우선
3. **TTL > retry window** — provider가 48시간 재시도하면 dedup cache도 48시간 이상 유지
4. **Storage 선택**: Redis/Memcached (TTL native), DB (transactional 필요시)

### 10.2 DS-APM 권장 idempotency key 스키마

```
ds_apm_idem_key = sha256(
  alert.fingerprint ||
  channel.id ||
  dispatch.round_no
)
```

- `alert.fingerprint` — Alertmanager가 준 것 그대로
- `channel.id` — Slack/Teams/PD/Email 각각 분리 (한 알람을 4채널로 보낼 때 4개 key)
- `dispatch.round_no` — replay 시 명시적으로 증가 (운영자가 "다시 보내기" 누른 횟수)

### 10.3 Retry & DLQ 정책

- 2xx만 success
- Exponential backoff + **jitter** (thundering herd 방지)
- 4xx (429 제외)는 즉시 DLQ
- DLQ 엔트리는 **원본 payload + 모든 시도 timestamp + 응답 코드 + 에러 메시지** 보존
- 운영자 manual replay 가능해야 함 → DLQ UI 또는 CLI 필요

---

## 11. Use Case 다이어그램 표준 — 무엇을 그릴 것인가

### 11.1 결론: **swimlane sequence + state machine 조합**

| 다이어그램 | 용도 | DS-APM 적용 |
|---|---|---|
| **Swimlane Sequence (UML)** | actor 여러 명의 시간 순 협업을 보일 때. STEP(Sequentially Timed Events Plotting) 기법과 동일. | Alert → SOP grounding → AI draft → Operator approval → Dispatch → Channel ack 흐름 (5 lanes) |
| **State Machine** | 한 entity가 거치는 상태 전이 | Alert lifecycle: `received → grounded → draft_pending → approved → dispatching → delivered / failed_dlq / replayed` |
| **Activity Diagram (decision branching)** | 분기 많은 happy/sad path 한 장에 | runbook validation failure, LLM auth failure 같은 sad path |

### 11.2 DS-APM Use Case용 swimlane sequence 권장 lanes

1. **SigNoz / Alertmanager** (이벤트 발생원)
2. **DS-APM Ingestion** (webhook 수신, PII redaction)
3. **AI Engine** (SOP grounding, draft 생성)
4. **Operator** (검수, approve/reject)
5. **Dispatcher** (channel adapter, retry, DLQ)
6. **External Channel** (Slack/Teams/PD/Email)

→ Happy path 1장 + Sad path별 1장씩 (LLM auth fail, runbook validation fail, channel 5xx + DLQ enqueue) 권장.

---

## 12. DS-APM Use Case 적용 템플릿 (베껴쓰기용)

### 12.1 Use Case 1건 표준 구조

```markdown
## UC-{NN}: {유스케이스 이름}

**Actor (Primary)**: 운영자 (Operator)
**Actor (Supporting)**: SigNoz, DS-APM Ingress, AI Engine, Dispatcher, Slack/MSTeams/PagerDuty/Email
**Trigger**: {이벤트 — 예: SigNoz가 payment-api 5xx 알람 firing}
**Preconditions**:
  - SigNoz alert rule이 활성화돼 있음
  - 대상 채널이 등록·healthy 상태
  - 대응 SOP가 vector store에 인덱싱돼 있음

**Postconditions (Success)**:
  - 알람이 승인된 runbook draft와 함께 채널에 전달됨
  - Audit log에 dispatch 기록 (idempotency key 포함) 남음

**Postconditions (Failure)**:
  - 실패한 dispatch는 DLQ에 enqueue되고 replay 가능
  - 운영자에게 fallback 알람이 발송됨

**Main Flow (Happy Path)**:
  1. SigNoz가 webhook으로 alert 페이로드(POST) 전송
  2. DS-APM Ingress가 fingerprint 추출 + PII redactor 통과
  3. AI Engine이 annotations.runbook_url 기반 SOP retrieval
  4. AI Engine이 draft 생성 (모델, prompt template, citation 포함)
  5. 운영자에게 draft 검수 요청 알림
  6. 운영자 approve
  7. Dispatcher가 채널별 adapter로 payload 변환 + 전송
  8. 채널 2xx 응답 → audit log

**Alternate Flow A — LLM auth failure**:
  4a. AI Engine이 401/403 수신 → degraded mode (SOP 원문만 전달)
  4b. SRE에 LLM auth alert 발송

**Alternate Flow B — Runbook validation failure**:
  4c. SOP retrieval 0건 또는 staleness > 90d → draft 생성 보류, 운영자에 raw alert만 전달

**Alternate Flow C — Channel 5xx**:
  7a. Exponential backoff with jitter 재시도 (max N회)
  7b. 모두 실패 → DLQ enqueue (payload, attempts, last_error 보존)
  7c. fallback 채널로 운영자에게 fan-out

**Non-functional**:
  - Ingress → Dispatch p95 latency ≤ 30s
  - PII redaction 100% before AI Engine
  - Idempotency: 같은 (fingerprint, channel) 중복 dispatch 금지

**Diagrams**: §11의 swimlane sequence 1장 + state machine 1장
```

### 12.2 SOP / Runbook 1건 표준 구조 (DS-APM이 grounding할 단위)

```markdown
# Runbook: {제목}

**Owner**: {팀명 / Slack 채널}
**Last Updated**: 2026-05-28 (next review: 2026-08-28)
**Severity**: SEV-2
**Estimated Duration**: 15min
**Risk Level**: Medium (rollback 포함)
**Approval Required**: Yes (production deploy 변경 시)

## 1. Trigger
- Alert: `PaymentService5xxHigh`
- Symptom: payment-api 5xx error rate > 5% for 5m
- Indicators: SigNoz dashboard `payment-overview`, log query `service=payment-api status>=500`

## 2. Pre-checks
- [ ] payment-gateway upstream 헬스 확인
- [ ] 최근 30분 내 deploy 여부 확인 (`kubectl rollout history`)
- [ ] DB connection pool saturation 확인
- [ ] Recent config change 확인

## 3. Steps
1. payment-gateway latency 확인 (SigNoz trace `service.name=payment-gateway`)
2. payment-gateway가 unhealthy → §3a
3. 최근 deploy SHA가 의심 → §3b
4. DB pool 포화 → §3c

### 3a. Gateway 우회
```bash
kubectl set env deployment/payment-api PAYMENT_GATEWAY_URL=https://backup-gw.example.com
```

### 3b. Rollback
```bash
kubectl rollout undo deployment/payment-api
```

### 3c. Pool 확장
```bash
kubectl set env deployment/payment-api DB_POOL_SIZE=50
```

## 4. Verification
- Error rate < 0.5% for 10m
- p95 latency < 500ms
- SigNoz alert `PaymentService5xxHigh`가 resolved 상태로 전환

## 5. Rollback
- §3a 적용 후 문제 지속 → 원래 URL로 복원, §3b로 진행
- §3b 적용 후 새 문제 발생 → `kubectl rollout undo` 한 번 더

## 6. Escalation
- 15분 내 미해결 → @payments-oncall
- 30분 내 미해결 → IC 선언, Deputy 지명, @cto

## 7. SLA
- Detection → Ack: 5min
- Ack → Mitigation: 15min
- Postmortem due: SEV-2이므로 5 business days

## 8. Related Alerts
- `PaymentGatewayDown`
- `PaymentDBPoolSaturated`

## 9. Postmortem Links
- (이 runbook으로 해결된 과거 incident 링크)
```

---

## 13. 기능명세서에 들어갈 도메인 표준 필드 목록

### 13.1 Alert 객체 (DS-APM Ingress가 받는 표준 스키마)

| 필드 | 출처 표준 | DS-APM 필수성 |
|---|---|---|
| `alertname` | Prometheus | 필수 |
| `status` (`firing`/`resolved`) | Alertmanager | 필수 |
| `severity` (`critical`/`error`/`warning`/`info`) | PagerDuty + Prometheus 합집합 | 필수 |
| `urgency` (`high`/`low`) | PagerDuty / ITIL | 권장 (severity와 분리) |
| `impact` (`catastrophic`/`major`/`minor`/`trivial`) | ITIL | 선택 |
| `priority` (P1~P4) | ITIL | 선택 (severity로부터 derive 가능) |
| `startsAt`, `endsAt` | Alertmanager | 필수 |
| `labels.service`, `labels.team`, `labels.deployment_environment` | OTel Resource | 필수 |
| `labels.{k8s.cluster, k8s.namespace, k8s.pod}` | OTel Resource | 환경별 필수 |
| `annotations.summary` | Prometheus 관례 | 필수 |
| `annotations.description` | Prometheus 관례 | 필수 |
| `annotations.runbook_url` | Prometheus 관례 | **필수** (DS-APM grounding 1차 키) |
| `annotations.dashboard_url` | 관례 | 권장 |
| `generatorURL` | Alertmanager | 필수 |
| `fingerprint` | Alertmanager | **필수** (idempotency 시드) |
| `correlation_id` | DS-APM 자체 | 필수 (incident 묶음 식별) |
| `trace_id` (있을 시) | OTel | 권장 |

### 13.2 SOP / Runbook 객체

| 필드 | 비고 |
|---|---|
| `runbook_id` (`rb_*`) | 고유 ID |
| `title` | |
| `owner_team`, `owner_channel` | GitLab/Atlassian 관례 |
| `severity_target` | 매칭되는 alert severity |
| `last_updated`, `next_review_due` | 90일 review SLA 권장 |
| `trigger.alertname[]` | 다대다 매칭 가능 |
| `pre_checks[]` | 체크리스트 |
| `steps[]` (each: title, command, expected_output, risk) | |
| `verification[]` (each: metric, threshold) | 정량 verification 필수 |
| `rollback[]` | |
| `escalation_matrix` | 시간 임계치별 |
| `sla.detection_to_ack`, `sla.ack_to_mitigation` | |
| `related_alerts[]` | |
| `staleness_days` (>90 → block draft) | DS-APM 자체 검증 |

### 13.3 Draft 객체 (AI Engine 출력)

| 필드 | 비고 |
|---|---|
| `draft_id` | |
| `alert_fingerprint` | 원본 alert 참조 |
| `runbook_ids[]` | grounding된 SOP들 |
| `model.name`, `model.version`, `model.temperature` | reproducibility |
| `prompt_template_id`, `prompt_template_version` | |
| `citations[]` (each: runbook_id, section, confidence) | |
| `body_markdown` | |
| `created_at` | |
| `approval_status` (`pending`/`approved`/`rejected`/`expired`) | |
| `approved_by`, `approved_at`, `rejection_reason` | |
| `redaction_applied` (boolean + categories[]) | PII 정책 audit |

### 13.4 Dispatch 객체

| 필드 | 비고 |
|---|---|
| `dispatch_id` | |
| `idempotency_key` (sha256(fingerprint, channel_id, round_no)) | |
| `alert_fingerprint`, `draft_id` | 참조 |
| `channel.type` (`slack`/`msteams`/`pagerduty`/`webhook`/`email`) | |
| `channel.id`, `channel.target` | |
| `payload_rendered` (channel별 adapter 결과) | |
| `attempt_no`, `max_attempts` | |
| `last_attempt_at`, `last_status_code`, `last_error` | |
| `state` (`pending`/`sent`/`delivered`/`failed`/`dlq`) | |
| `dlq_enqueued_at` (nullable) | |
| `replay_of_dispatch_id` (nullable) | manual replay 추적 |

### 13.5 Postmortem 객체 (DS-APM이 트리거하는 후속 자료)

PagerDuty 템플릿 그대로 채택 권장 (§2.5 참조).

---

## 14. DS-APM 산출 질문에 대한 직접 답변

### Q1. Incident → SOP → handoff 흐름의 표준 다이어그램은?

**답**: Swimlane sequence diagram이 표준. 5~6 lane (Monitoring source / Ingress / AI Engine / Operator / Dispatcher / Channel). State machine을 보조로 추가 (alert lifecycle). Sad path는 activity diagram 분기로 별도 작성. STEP(Sequentially Timed Events Plotting) 기법이 incident investigation의 정설.

### Q2. Runbook 1건의 권장 구조?

**답**: 13개 필드 표준 (§12.2). Title / Owner / Severity / Trigger / Pre-checks / Steps / Verification / Rollback / Escalation / SLA / Related Alerts / Last Updated / Postmortem Links. GitLab은 "as short as possible, complete enough to be executed without further research"를 원칙으로 함.

### Q3. 채널 페이로드 명세를 기능명세에 어떻게 박나?

**답**: 3단 구조 권장.
1. **Channel-agnostic dispatch schema** (DS-APM 내부 canonical payload)
2. **채널별 adapter 매핑 표** (canonical field → Slack block / Teams card / PD field)
3. **각 채널별 1건 이상 실제 예시 페이로드** (§7, §6 참고)

매핑 표 예시:

| Canonical | Slack | MS Teams | PagerDuty |
|---|---|---|---|
| `title` | header block text | TextBlock (size=Large) | `payload.summary` |
| `severity` | header prefix (`[SEV-2]`) | TextBlock | `payload.severity` |
| `service` | section field | FactSet entry | `payload.component` |
| `body` | section mrkdwn | TextBlock wrap | `payload.custom_details.body` |
| `runbook_url` | actions button | Action.OpenUrl | `payload.custom_details.runbook_url` |
| `dedup_key` | (metadata) | (metadata) | `dedup_key` (top-level) |

### Q4. 알람 메타데이터 표준 필드?

**답**: §13.1 표 참조. **반드시 갖춰야 할 8개**: `alertname`, `severity`, `status`, `startsAt`, `labels.service`, `annotations.summary`, `annotations.runbook_url`, `fingerprint`.

### Q5. PII redaction 정책을 명세에 어떻게 기술하나?

**답**: 4단 구조.
1. **Categories** (이메일, 신용카드, 전화, IP, user_id 등) 명시적 enum
2. **Per-category strategy** (drop / hash / truncate / mask)
3. **Pipeline location** (instrumentation > collector > DS-APM ingress, 가장 이른 단계에서)
4. **Monitoring** (redaction rate > threshold → meta-alert)

OTel Collector의 4개 processor (Attributes / Filter / Redaction / Transform)에 매핑해서 어느 processor가 어느 category를 담당하는지 표로 박는다.

### Q6. DLQ + replay idempotency key 설계 관행?

**답**: §10.2의 `sha256(alert.fingerprint || channel.id || dispatch.round_no)`. 자연 키 활용, deterministic, TTL은 max retry window의 2배 권장. Storage는 Redis (TTL native).

---

## 15. 참고 링크 (공식 출처 우선)

### Google SRE
- [SRE Book — Practical Alerting](https://sre.google/sre-book/practical-alerting/)
- [SRE Book — Managing Incidents](https://sre.google/sre-book/managing-incidents/)
- [SRE Book — Effective Troubleshooting](https://sre.google/sre-book/effective-troubleshooting/)
- [SRE Book — Postmortem Culture](https://sre.google/sre-book/postmortem-culture/)
- [SRE Workbook — Incident Response](https://sre.google/workbook/incident-response/)

### PagerDuty
- [Incident Response Documentation](https://response.pagerduty.com/)
- [Severity Levels](https://response.pagerduty.com/before/severity_levels/)
- [Different Roles](https://response.pagerduty.com/before/different_roles/)
- [What is an Incident?](https://response.pagerduty.com/before/what_is_an_incident/)
- [During an Incident](https://response.pagerduty.com/during/during_an_incident/)
- [Postmortem Process](https://response.pagerduty.com/after/post_mortem_process/)
- [Postmortem Template](https://response.pagerduty.com/after/post_mortem_template/)
- [Events API v2 — Send an Alert Event](https://developer.pagerduty.com/docs/events-api-v2/trigger-events/index.html)

### GitLab
- [GitLab Runbooks](https://runbooks.gitlab.com/)
- [GitLab Docs — Runbooks](https://docs.gitlab.com/user/project/clusters/runbooks/)
- [GitLab Production Engineering Handbook](https://handbook.gitlab.com/handbook/engineering/infrastructure-platforms/production-engineering/)

### OpenTelemetry / SigNoz
- [OTel Signals Concept](https://opentelemetry.io/docs/concepts/signals/)
- [OTel Resource Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/resource/)
- [OTel Handling Sensitive Data](https://opentelemetry.io/docs/security/handling-sensitive-data/)
- [SigNoz Alerts](https://signoz.io/docs/alerts/)
- [SigNoz Alerts Management](https://signoz.io/docs/userguide/alerts-management/)

### Prometheus / Alertmanager
- [Prometheus Alerting Best Practices](https://prometheus.io/docs/practices/alerting/)
- [Alertmanager Webhook Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Alertmanager Notifications Reference](https://prometheus.io/docs/alerting/latest/notifications/)

### Atlassian / AWS / Microsoft
- [Atlassian Incident Management Handbook](https://www.atlassian.com/incident-management/handbook)
- [Atlassian Incident Management Hub](https://www.atlassian.com/incident-management)
- [AWS Well-Architected — Operational Excellence Pillar](https://docs.aws.amazon.com/wellarchitected/latest/operational-excellence-pillar/welcome.html)
- [MS Teams — Adaptive Cards Overview](https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/universal-actions-for-adaptive-cards/overview)
- [Slack — Sending Messages Using Incoming Webhooks](https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks)

### ITIL
- [PagerDuty — Incident Priority Matrix](https://www.pagerduty.com/resources/digital-operations/learn/incident-priority-matrix/)
- [ITIL 4 Priority Matrix (PDCA Consulting)](https://pdcaconsulting.com/itil-priority-matrix-templates-incident-problem-request/)

### Webhook / DLQ / Idempotency
- [Hookdeck — Implementing Webhook Idempotency](https://hookdeck.com/webhooks/guides/implement-webhook-idempotency)
- [Hookdeck — Outbound Webhook Retry Best Practices](https://hookdeck.com/outpost/guides/outbound-webhook-retry-best-practices)
- [Alertmanager Webhook Payload Example (gist)](https://gist.github.com/mobeigi/5a96f326bc06c7d6f283ecb7cb083f2b)

### Diagramming
- [Humanreliability — Swimlanes & STEP](https://www.humanreliability.com/2024/06/incident-investigation-swimlanes-sequentially-timed-events-plotting-step/)
- [Wikipedia — Swim Lane](https://en.wikipedia.org/wiki/Swim_lane)
