---
id: SPEC-INDEX
title: DS-APM 기능명세서 (AI-Ops 전략 → 기능분해)
type: srs-index
template: BMAD PRD(Vision→JTBD→UJ→Features→FR) × Spec by Example
status: draft
source_of_truth: ../_foundation/source-strategy-brief.md
decomposition: top-down (사업전략 → 고객 voice FR → 코드 근거)
scope: hybrid (구현 핵심 CF-1~6 상세 + 로드맵 CF-7~10 저fidelity)
maps_modules: [F0, F1, F2, F3, F4, F5, F6, F7, F8]
updated: 2026-06-08
---

# DS-APM 기능명세서 (SRS)

> **분해 방향**: [`사업 전략서`](../_foundation/source-strategy-brief.md)에서 **top-down** — Vision → Target User·Jobs → User Journey → Feature → FR. FR은 **고객/운영 관점**("[운영자/AM/보안]는 ~한다")으로 쓰고, 코드 메커니즘은 §7 Coverage Map의 *구현 근거* 열로 강등한다.
> **범위(하이브리드)**: 실제 코드로 구현된 핵심(CF-1~6)은 FR 상세 + Given/When/Then까지. 전략 로드맵의 미구현 역량(CF-7~10)은 저fidelity FR + 단계 태그만.
> **표기**: FR-CFn.m, NF-x.y, SM-n, UJ-n, JTBD-n. Acceptance는 Gherkin(영문, godog).

---

## §0. Document Purpose

본 문서는 *AI-Ops 기반 지능형 관제 체계 고도화 및 운영 조직 혁신 전략*(착수: 26'2Q, 김동우·이준경·박진혁)의 **기능 요구를 분해**한 것이다. 전략의 정성 목표("PM→AM 전환", "운영 상향평준화", "지식 자산화")를 운영자·관리자·보안 관점의 검증 가능한 FR로 변환하고, 각 FR을 구현 코드(또는 로드맵 단계)에 매핑한다.

독자: 경영/발주(목표·성과 SM 확인) · AM/PM(역할 전환 범위) · 운영자(1차 대응 절차) · 보안/감사(격리·비노출·추적) · 개발자(구현 근거·인수조건).

---

## §1. Vision

**운영을 개인의 경험에서 분리해 플랫폼에 내재화한다.** 관측 알람을 운영 절차서(SOP)에 자동 연계하고, AI가 SOP에 근거한 대응 가이드를 만들어 운영자에게 넘긴다. 목표는 셋이다.

1. **PM의 역할 확장(To-AM)** — 저부가가치 기술 운영(Ops)을 AI·시스템에 이관, PM을 고객가치·비즈니스 확장의 AM으로 전환.
2. **운영 조직의 상향 평준화** — SOP 기반 조치 가이드로 **초·중급 인력만으로 인시던트 1차 대응**.
3. **기술 자산화** — 자동 보고서·스펙↔코드 연결로 운영 지식을 시스템에 내재화.

운영 1원칙: **"완전 자동화가 아닌 지능형 자동화 + Human-in-the-loop."** silent drop(알람 누락)은 AI 부정확함보다 나쁘다.

---

## §2. Target Users & Jobs-to-be-Done

| Persona | 현재 고통 | Job (해결하려는 것) |
|---|---|---|
| **PM/PL → AM** | 대형 장애 연 2~3건뿐인데 상시 모니터링에 매일 1~2시간 고정 낭비; 인시던트마다 로그·트랜잭션 직접 대조 | 저부가가치 Ops에서 해방되어 고객가치·비즈니스로 이동 |
| **운영자 (초/중급, L1)** | 전문가(특정 PM/PL) 경험에 의존, 담당자 부재 시 대응 불가 | SOP 가이드로 **전문가 없이 1차 대응** |
| **운영 조직 / 경영** | 특정 인력 의존, 인력 변동에 취약, 지식 개인 편중 | 자생적 대응력 + 연 50% 리소스 세이브 + lock-in |
| **보안 / 감사** | (혁신 도입 시 신규 리스크) 민감정보·테넌트 격리·추적 | 외부 비노출·격리·전 행위 추적 보장 |

**Jobs-to-be-Done (FR의 출발점):**
- **JTBD-1** 이슈 없는 알람·점검에 매일 묶이고 싶지 않다 → 상시 모니터링 자동화
- **JTBD-2** 인시던트 원인·대응 가이드를 직접 대조 없이 받고 싶다 → 반자동 RCA + AI 초안
- **JTBD-3** 전문가 없이 초·중급 인력이 1차 대응하게 하고 싶다 → SOP 조치 가이드 + 핸드오프
- **JTBD-4** AI가 틀려도 잘못된 자동조치로 장애가 커지면 안 된다 → HITL·승인·rollback·fail-open
- **JTBD-5** 장애 보고서·운영 지식을 손으로 안 만들고 자산화하고 싶다 → 자동 보고서·RCA 템플릿·지식화
- **JTBD-6** 개인 경험이 아닌 플랫폼에 운영을 내재화하고 싶다 → 표준화·정책·감사 기반

---

## §3. Success Metrics

전략 최종 목표는 연간 운영 리소스 **50% 세이브(324h→162h)** 이며, 이는 로드맵 완성 시 달성된다(§6·§9.3). 본 기능명세의 **구현 핵심(CF-1~6)이 직접 기여**하는 지표는 다음과 같다.

| SM | 지표 | 기존 | 목표 | 기여 Feature |
|---|---|---|---|---|
| SM-2 | 상시 확인 업무 | 240h | 120h | CF-1·2·3 |
| SM-3 | 인시던트 분석(10건) | 40h | 20h | CF-2 |
| **SM-C1** *(counter)* | 알람/SOP 정보 silent drop | — | **0건** | NF-5.2.* |
| **SM-C2** *(counter)* | AI 오탐 기반 오대응 | — | **0건** | FR-CF2.3 (HITL) |

> 장애 보고 자산화·장애처리 자동화에 따른 추가 절감(전략서 §5: 보고 24→12h, 장애처리 20→10h)은 **먼 미래 로드맵**이므로 §9.3에만 둔다.

---

## §4. Product Perspective & Operating Environment

**AIOpsAgent**는 SigNoz **community 빌드**(OTel-native 관측 OSS) 위에 *Incident → SOP → AI 가이드 → Operator handoff* 레이어를 얹은 확장이다(fork 아님). 관측·수집 자체(OTel, ClickHouse, 대시보드)는 **SigNoz upstream의 책임 = 전제 환경**이며 본 명세의 FR 대상이 아니다.

| 의존 | 역할 | 본 명세 |
|---|---|---|
| SigNoz Ruler / Alertmanager | alert 평가·firing·dispatch hot path | CF-3·5가 wrapping |
| OTel Collector / ClickHouse | 로그·메트릭·트레이스 수집·저장 | **전제(upstream)** — Non-goal §9 |
| PostgreSQL (bun ORM) | SOP store (마이그레이션 078 `ds_sop_documents`) | CF-1 |
| LLM Provider (Claude/Codex) | HTTP/JSON, 401/403/429/5xx 표준 | CF-2 |

제약: Go 단일 binary `cmd/community/` · Enterprise 모듈(`ee/`) 불변 · multi-tenant/PII는 production-ready 아님(README) · y2i 영구 비활성.

---

## §5. User Journeys

운영 가치 사슬(관제 → 분석/지식화 → ITSM)의 핵심 여정. **별도 use-case 문서 없이 본 PRD에 내장**한다(BMAD UJ). 행위·인수조건 상세(Given/When/Then)는 각 CF feature 파일.

### UJ-1 골든패스 (정상) — CF-1·2·3·4·6
- **트리거**: 결제 서비스 5xx 비율 임계 초과 → 알람 firing. **Persona**: 운영자.
- **경로**: 테넌트 식별 → SOP 자동 연계(CF-1) → PII 마스킹(CF-4) → AI 대응 가이드 생성(CF-2) → 5채널 핸드오프(CF-3) → 운영자 검수(approved) → 감사 기록(CF-6).
- **분기**: SOP 미연계/테넌트 불일치/비활성 → 원본 알람만 · AI 실패 → UJ-3 · 채널 실패 → UJ-2.
- **결과**: 운영자가 SOP+AI 가이드 알림 수령·검수. **최소보장**: 어떤 분기든 원본 알람은 전달(silent drop 0).

### UJ-2 실패·복구 (DLQ) — CF-3·5·6
- **트리거**: 채널 전송 terminal failure(2xx 아님 + 취소 아님).
- **경로**: 실패 건 DLQ 보존(CF-5, 채널별 entry) → dispatcher 무중단 → 운영자 재발송 → ledger 멱등 확인 → 신규 재전송/기존 skip → 감사.
- **분기**: context 취소 → 정상 종료(DLQ 미적재) · `dlqSink=nil` → 로그만(기본 배선 open) · 재전송 또 실패 → 새 entry.
- **결과**: 실패 알림 무유실 보존 + 중복 없는 재발송. (HMAC 서명 open.)

### UJ-3 degraded (LLM fail-open) — CF-2·6
- **트리거**: AI 가이드 생성 중 LLM 401/403/429/timeout 또는 제어 위반.
- **경로**: dispatch hook이 error 미반환·1초 budget으로 입력 그대로 통과(CF-2 fail-open) → 가이드 안전 degrade(상태·사용량 감사) → 운영자는 알람+SOP 원문 수령(AI 요약 제외) → fail-open 감사(CF-6).
- **결과**: AI 부재에도 정보 손실 0. AI 실패는 silent drop 아닌 fallback/meta-alert로 발화.

### UJ-4 사전대응·자산화 *(로드맵)* — CF-7·8·9
이상 징후 사전 경고(CF-7) → 승인 기반 자동조치(CF-8) → 자동 보고서·지식 자산화(CF-9). 미구현(§6·§9.3).

---

## §6. Feature Map

incident 생명주기 순. **CF-1~6 = 구현 핵심(★)**, **CF-7~10 = 전략 로드맵(○ 미구현)**.

| CF | Feature (사용자 가치) | JTBD | 상태 | 단계 | 구 모듈 | 파일 |
|---|---|---|---|---|---|---|
| **CF-1** | SOP 자동 연계 (Grounding) + 테넌트 격리 | 1,3 | ★ implemented | 1~2단계 | F1·F4 | [features/CF-1-grounding.md](features/CF-1-grounding.md) |
| **CF-2** | AI 대응 가이드 + 안전(HITL·fail-open) | 2,3,4 | ★ implemented | 2단계 | F2·F3 | [features/CF-2-ai-assist.md](features/CF-2-ai-assist.md) |
| **CF-3** | 멀티채널 핸드오프 (5채널) | 1,3 | ★ implemented | 2단계 | F6 | [features/CF-3-handoff.md](features/CF-3-handoff.md) |
| **CF-4** | 민감정보 비노출 (PII Safety) | 4 | ★ implemented | risk | F7 | [features/CF-4-pii-safety.md](features/CF-4-pii-safety.md) |
| **CF-5** | 무유실·멱등 재처리 (DLQ) | 4 | ★ implemented-mvp | risk | F8 | [features/CF-5-reliable-delivery.md](features/CF-5-reliable-delivery.md) |
| **CF-6** | 정책·감사 기반 (Foundation·Audit) | 6 | ★ implemented | 1단계 | F0·F5 | [features/CF-6-foundation-audit.md](features/CF-6-foundation-audit.md) |
| **CF-7** | 이상 탐지 (Anomaly Detection) | 1 | ○ planned | 2단계 | — | (로드맵) |
| **CF-8** | 자동 조치 (Auto-Remediation, 승인 기반) | 4 | ○ planned | 3단계 | — | (로드맵) |
| **CF-9** | 지식 자산화·자동 보고서 (LLM Wiki·RCA 템플릿) | 5 | ○ planned | 27'1Q | — | (로드맵) |
| **CF-10** | ITSM 통합·Workflow | 6 | ○ planned | 27'3Q | — | (로드맵) |

---

## §7. Requirements Coverage Map

### §7.1 구현 핵심 (CF-1~6) — 고객 voice FR + 구현 근거

| FR | 요구 (고객 voice) | JTBD | UJ | WBS |
|---|---|---|---|---|
| FR-CF1.1 | 운영자는 알람이 뜨면 거기 연결된 대응 절차서(SOP)를 자동으로 함께 받는다 — 어느 SOP인지 직접 찾지 않는다 | 1 | UJ-1 | WBS-1.1 |
| FR-CF1.2 | 관리자는 팀 SOP를 등록해 두면 시스템이 보관하고 알람에 매칭함을 신뢰한다 | 6 | UJ-1 | WBS-1.1 |
| FR-CF1.3 | 보안담당자는 한 팀 운영자가 다른 팀 SOP에 접근하지 못하도록 격리를 보장받는다 | 4 | UJ-1 | WBS-1.0 |
| FR-CF1.4 | 보안담당자는 다른 팀 SOP의 *존재 여부조차* 노출되지 않음을 보장받는다 | 4 | UJ-1 | WBS-1.0 |
| FR-CF1.5 | 운영자는 비활성·만료(90일+) SOP는 적용되지 않고 원본 알람만 받음을 안다 | 3 | UJ-1 | WBS-1.1 |
| FR-CF2.1 | 운영자는 알람마다 SOP에 근거한 대응 가이드(원인 가설·첫 조치·고객/벤더 안내 초안)를 AI로부터 받는다 | 2 | UJ-1 | WBS-1.2 |
| FR-CF2.2 | 초·중급 운영자는 전문가 없이도 이 가이드로 1차 대응을 시작할 수 있다 | 3 | UJ-1 | WBS-1.2 |
| FR-CF2.3 | 운영자는 AI가 "자동 조치했다"고 주장하지 않으며 모든 조치에 사람 승인이 필요함을 보장받는다 | 4 | UJ-1 | WBS-1.2 |
| FR-CF2.4 | 운영자는 AI가 느리거나 실패해도 알람을 지연·누락 없이 받는다 (fail-open) | 4 | UJ-3 | WBS-1.2 |
| FR-CF2.5 | 관리자는 AI 사용량/예산을 초과해도 알람 전달이 막히지 않게 제어한다 | 4 | UJ-3 | WBS-1.2 |
| FR-CF2.6 | 운영자는 동일 장애 재발 시 과거 대응 이력을 참조할 수 있다 | 2 | UJ-1 | WBS-1.2 |
| FR-CF3.1 | 운영자는 SOP·AI 가이드가 포함된 알림을 평소 쓰는 채널(Slack·Teams·PagerDuty·Webhook·Email)로 받는다 | 1,3 | UJ-1 | WBS-1.3 |
| FR-CF3.2 | 관리자는 알림에 표시할 항목(서비스·SOP·AI 요약 등)을 템플릿으로 정의할 수 있다 | 6 | UJ-1 | WBS-1.3 |
| FR-CF3.3 | 운영자는 한 채널이 실패해도 다른 채널·후속 처리가 멈추지 않음을 보장받는다 | 4 | UJ-2 | WBS-1.3 |
| FR-CF4.1 | 보안담당자는 외부 채널로 나가는 메시지에 이메일·전화·토큰·비밀번호가 노출되지 않음을 보장받는다 | 4 | UJ-1 | WBS-1.4 |
| FR-CF5.1 | 운영자는 채널 전송이 최종 실패한 알림이 유실되지 않고 보관됨을 보장받는다 | 4 | UJ-2 | WBS-1.5 |
| FR-CF5.2 | 운영자는 실패 알림 재발송 시 같은 알림이 중복으로 두 번 가지 않음을 보장받는다 | 4 | UJ-2 | WBS-1.5 |
| FR-CF5.3 *(open)* | 보안담당자는 재발송 페이로드 위변조 방지(HMAC)를 요구한다 — **정책 미정** | 4 | UJ-2 | WBS-1.5 |
| FR-CF6.1 | 관리자는 팀별 SOP·AI·감사 정책을 설정으로 관리한다 | 6 | UJ-1 | WBS-1.0 |
| FR-CF6.2 | 감사담당자는 SOP 접근·AI 호출·전달 1건마다 감사 기록이 남음을 보장받는다 | 6 | UJ-1·2·3 | WBS-1.0 |
| FR-CF6.3 | 감사담당자는 감사 기록 실패가 서비스 부팅·운영을 막지 않음을 보장받는다 | 6 | UJ-1 | WBS-1.0 |

> 코드 함수·상수·라벨명 등 *구현 메커니즘*은 각 feature 파일의 `구현 근거` 절에 둔다(예: `PreviewSOPDocumentBinding`, `ErrSOPDocumentNotFound`, `MarkIfNew`, `[redacted]`). 공개 repo는 squash라 개별 커밋이 없어 코드 경로로 추적한다.

### §7.2 로드맵 (CF-7~10) — 저fidelity FR + 선행

| FR | 요구 (고객 voice) | JTBD | 단계 | 선행 |
|---|---|---|---|---|
| FR-CF7.1 | 운영자는 비정상 임계치를 사전 경고로 받는다 (AI 기준선 학습) | 1 | 2단계 | OTel·데이터 학습 |
| FR-CF8.1 | 운영자는 정형 장애에 **승인 기반** 자동 조치 스크립트를 적용할 수 있다 (rollback 포함) | 4 | 3단계 | CF-2·5, Runbook 검증 |
| FR-CF9.1 | AM은 장애 보고서가 자동 생성되고 Incident가 구조화·라벨링됨을 받는다 | 5 | 27'1Q | RCA 템플릿·데이터 정교화 |
| FR-CF9.2 | 운영 조직은 운영 지식이 LLM Wiki로 자산화·스펙↔코드 연결됨을 받는다 | 5 | 27'1Q | CF-9.1 |
| FR-CF10.1 | 운영 조직은 OSS ITSM 워크플로로 운영 자동화·지속 개선 구조를 갖춘다 | 6 | 27'3Q | CF-9 |

---

## §8. Cross-cutting Non-functional Requirements

전략 §6 리스크에서 유도한 것 포함.

### §8.1 Performance
- **NF-5.1.1** webhook 수신 → 채널 2xx 응답 p95 ≤ 30초 (운영자 approve 시간 제외).
- **NF-5.1.2** dispatcher hot path AI 가이드 생성 p95 ≤ 1초.

### §8.2 Safety (전략 리스크: AI 오탐 / 자동조치 실패)
- **NF-5.2.1** LLM 실패는 silent drop이 아닌 fallback dispatch 또는 meta-alert로 발화. (→ SM-C1)
- **NF-5.2.2** AI 도달 전 PII(email·phone·16자+ secret) 100% redaction.
- **NF-5.2.3** Audit sink 실패는 서버 부팅을 막지 않는다.
- **NF-5.2.4** 채널 dispatch 실패는 dispatcher를 중단시키지 않는다.
- **NF-5.2.5** AI는 자동 실행을 주장하지 않으며 모든 조치는 Human-in-the-loop 승인. (→ SM-C2, 리스크 "AI 오탐")
- **NF-5.2.6** *(로드맵)* 자동 조치는 rollback·승인 기반으로만 실행. (리스크 "자동조치 실패", CF-8)

### §8.3 Security
- **NF-5.3.1 (HMAC — OPEN)** Replay payload는 HMAC 서명되어야 한다. **정책 미정** → FR-CF5.3.
- **NF-5.3.2** outbound 채널 호출은 TLS 1.2+.
- **NF-5.3.3** Secret 자격증명은 contract response에 비노출.
- **NF-5.3.4** Cross-tenant lookup은 존재를 누설하지 않는다. → FR-CF1.4.

### §8.4 Quality / Data (전략 리스크: 알람 품질 / 데이터 품질)
- **NF-5.4.1** 재시도 전부 실패해도 원본 페이로드·시도 이력은 DLQ 보존.
- **NF-5.4.2** 동일 `(fingerprint, channel)` 중복 dispatch 0건.
- **NF-5.4.3** Dispatch/draft/SOP 접근/fail-open 1건당 audit row ≥ 1.
- **NF-5.4.4** *(로드맵)* Alert 튜닝·기준 재정의로 알람 품질 유지; 로그 표준화·정제로 자동화 정확도 확보.

### §8.5 Business Rules
- **NF-5.5.1** 운영자 알람에서 alert payload + SOP 본문 정보 손실 0건.
- **NF-5.5.2** Audit log는 reproducibility 용도. 개인 책임 추궁 금지.
- **NF-5.5.3** SOP `staleness_days` 90일 초과 시 grounding 보류, raw alert만 전달. → FR-CF1.5.

---

## §9. Non-goals · Roadmap · Drift

### §9.1 Out of Scope (영구)
- SigNoz upstream 자체 기능 + OTel/ClickHouse 수집·저장(전제 환경) · Enterprise 모듈(`ee/`)
- Vector retrieval SOP grounding (현재 explicit-label only) · Redis idempotency · y2i (영구 비활성)

### §9.2 Production-readiness 격차 (README)
- **Multi-tenant**: DB row-level security 없음 — label filter 수준, `project_id` spoofing 방어 없음. → CF-1
- **PII**: AIOpsAgent ingress 단일 지점 — instrumentation·OTel Collector 단계 미적용. → CF-4

### §9.3 Roadmap / Open (미구현)
- **CF-7~10** 전체 (이상탐지·자동조치·자동보고서/지식화·ITSM) — 단계 §6 참조.
- **로드맵 추가 절감 (전략서 §5, 먼 미래)**: 장애 보고/자산화 24→12h (CF-9), 장애처리 20→10h (CF-8). 전체 50% 세이브는 이 단계 완성 시 달성 — 현재 기능명세는 SM-2·SM-3만 직접 기여(§3).
- **HMAC 정책** (NF-5.3.1, FR-CF5.3) — replay 서명 미정.
- **DLQ 기본 배선** — 적재 로직은 있으나 `server.go` sink 주입이 기본 `nil`(미연결). → CF-5
- **Idempotency 키 확장** — `EventID = fingerprint`만; 권장 `sha256(fingerprint‖channel‖round)`. → CF-5
- **Replay API/UI**, **Redaction rate metric**, **Frontend 운영자 검수 화면** — 미구현.

### §9.4 Drift 정정 (component-source-map.md §9, 코드 검증)
- `sop_document_file_store.go` — 코드에서 삭제됨. SOP 영속화는 DB store. → CF-1.
- `opsgenie` — 보강 없는 6번째 채널. "5채널"은 AI/SOP 보강 채널 기준. → CF-3 각주.
- SOP delete 엔드포인트 미노출 (`SOPStore.Delete`는 존재). → CF-1.

---

## §10. References
- [`../_foundation/source-strategy-brief.md`](../_foundation/source-strategy-brief.md) — ★ 원본 사업 전략서 (top-down 진실의 원천)
- [`../_foundation/baseline.md`](../_foundation/baseline.md) — 11 commits / +12.6k LOC, 커밋↔모듈
- [`../_shared/component-source-map.md`](../_shared/component-source-map.md) — 6컴포넌트↔코드↔F0~F8, drift
- [`../_shared/traceability.md`](../_shared/traceability.md) — UC×F×WBS (**CF 축 재매핑 예정 — Step C**)
- BMAD-METHOD `bmm-skills/2-plan-workflows/bmad-prd` + `3-solutioning/bmad-create-epics-and-stories`
