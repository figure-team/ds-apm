---
id: BRIEF-OVERVIEW
title: DS-APM 시스템 개요 (요약본)
type: brief-overview
audience: 요약본 (의사결정용)
status: draft
source_artifact: OVERVIEW
links_to: [BRIEF, BRIEF-USECASE, BRIEF-SPEC, BRIEF-WBS, OVERVIEW]
updated: 2026-06-02
---

# DS-APM 시스템 개요 (요약본)

> **대상**: 팀장 · 매니저 · 의사결정자.
> **목적**: 상세본 Overview(아키텍처 표준 템플릿 arc42 v9.0)를 비개발자 관점으로 압축하여 비즈니스 가치, 시스템 경계, 위험, 아키텍처 결정 사항(ADR)만 정리합니다.
> **읽는 시간**: 약 10분. 기술 상세가 필요한 경우 본문 마지막의 상세본 산출물을 참조해 주십시오.

---

## 1. 개요

| 항목 | 요지 |
|---|---|
| **정체** | SigNoz Community 빌드의 알림 처리 경로에 운영 자동화(SOP 그라운딩·AI 초안·DLQ 재처리) 단계를 추가하는 확장 모듈 그룹 |
| **푸는 문제** | 운영 담당자가 새벽 알람을 받고 SOP(Standard Operating Procedure, 운영 절차서)를 찾아 협업 도구(메신저)로 발송하는 5~15분의 반복 업무를 30초 수준으로 단축 |
| **현 상태** | 착수 예정(planned) — 사전 PoC 검증 완료, 6개 구성 요소 착수 후 구축 예정 |
| **다음 결정** | 보안 정책(HMAC) · 다중 테넌트 격리 시점 · 개인 식별 정보 처리 단계 이동 |
| **다음 단계** | M-2 Production Readiness — 팀장 의사결정 후 확정 |

**AIOpsAgent**는 SigNoz Community 빌드의 알림 처리 경로에 운영 자동화(SOP 그라운딩·AI 초안·DLQ 재처리) 단계를 추가하는 확장 모듈 그룹입니다. SigNoz Alertmanager의 발송 핵심 경로(dispatcher hot path)에 AIOpsAgent 처리 단계를 삽입하여 **개인 식별 정보를 마스킹하고, 운영 절차서(SOP)를 자동 매칭하고, AI에게 초안을 작성시키고, 운영 담당자 검토를 거쳐 5채널 동시 발송**을 수행합니다. 운영 담당자의 핵심 책임(판단·승인)은 그대로 유지하고 반복 업무만 제거하는 구조입니다.

**7단계 처리 자동화 흐름**: 장애 감지 → 알람 수신 → 개인 식별 정보 마스킹 → SOP 매칭 → AI 초안 → 운영 담당자 승인 → 5채널 발송.

---

## 2. 시스템 경계 — 누구와 무엇을 주고받는가

AIOpsAgent는 SigNoz Community 빌드 안에 내장된 확장 모듈 그룹입니다. 따라서 "시스템 경계"는 **SigNoz 바이너리(AIOpsAgent 포함)의 바깥쪽** 기준으로 도식화합니다. 실제 외부와의 통신은 입구 1종(OTel/OTLP(OpenTelemetry/OpenTelemetry Protocol, 관측 표준) 텔레메트리)과 출구 2종(LLM(Large Language Model, 대규모 언어 모델) API, 5채널) 그리고 운영 담당자/SRE 인력뿐입니다. 그 외 모든 구성 요소 간 흐름은 같은 프로세스 안의 함수 호출입니다.

```mermaid
flowchart LR
    OTEL["외부 텔레메트리<br/>(OTel/OTLP)"]
    OP["운영 담당자<br/>(on-call)"]
    SRE["SRE 담당"]
    LLM["LLM Provider<br/>(외부 API)"]
    CH["5 채널<br/>(Slack/Teams/PD/Webhook/Email)"]

    subgraph SIGNOZ["SigNoz Community"]
        direction TB
        COLL["SigNoz Collector"]
        RULER["SigNoz Ruler"]
        AM["Alertmanager"]
        subgraph AIOPSAGENT["AIOpsAgent (확장 모듈 그룹)"]
            direction TB
            ING["Ingress"]
            PIIR["PII 마스킹 필터"]
            SOPENG["SOP 그라운딩 서비스"]
            AIENG["AI 초안 매니저"]
            DISP["알림 디스패처"]
            DLQ["DLQ 재처리 서비스"]
        end
        COLL --> RULER --> AM --> ING --> PIIR --> AIENG
        SOPENG -.-> AIENG
        AIENG --> DISP
        DISP --> DLQ
    end

    OTEL --> COLL
    OP <-->|승인| AIENG
    SRE <-.meta-alert.- DLQ
    AIENG -->|HTTPS| LLM
    DISP -->|HTTP/SMTP| CH
```

| 경계 | 방향 | 무엇을 주고받나 |
|---|---|---|
| 외부 → SigNoz 바이너리 (Collector) | 입력 | OTel/OTLP 텔레메트리 (외부 HTTP/gRPC) |
| SigNoz Alertmanager → AIOpsAgent | 내부 함수 호출 | 같은 프로세스 안의 함수 호출 (Webhook 아님) |
| SOP 그라운딩 서비스 ↔ AI 초안 매니저 | 내부 함수 호출 | 알람-SOP 자동 매칭 결과 |
| AIOpsAgent → LLM Provider | 출력 (외부) | SOP를 컨텍스트로 초안 요청 (HTTPS) |
| AIOpsAgent ↔ 운영 담당자 | 양방향 | 검수 요청 / 승인·반려 |
| AIOpsAgent → 5채널 | 출력 (외부) | 승인된 알림 동시 발송 (HTTP/SMTP) |
| AIOpsAgent → SRE | 알림 | AI 실패·DLQ(Dead Letter Queue, 미전송 사장 큐) 적재 등 메타 알람 |

**핵심**: 실제 외부 HTTP 호출은 **LLM API + 5채널** 2종에 한정됩니다. SigNoz Alertmanager → AIOpsAgent 사이는 같은 프로세스 안의 함수 호출이며 Webhook이 아닙니다.

**범위 밖**: SigNoz 자체 기능, SigNoz Enterprise 모듈, 외부 고객사용 별도 vector 검색 기능.

---

## 3. 내부 구성 — 6개 구성 요소

AIOpsAgent는 6개 모듈로 나뉘어 있습니다. 각 모듈은 독립된 책임을 가지며, 작업분해(WBS) 단위와 1:1로 매핑됩니다.

| 구성 요소 | 책임 | 비유 |
|---|---|---|
| **공통 기반 모듈 (Foundation Core)** | 공통 타입·감사 기록·테넌트 정책 검증 | 빌딩의 기초·골조 |
| **SOP 그라운딩 서비스 (SOP Grounding Service)** | 운영 절차서(SOP) 저장 + 알람-SOP 자동 매칭 | 사서·라이브러리 |
| **AI 초안 매니저 (AI Drafter Manager)** | LLM 초안 생성 + 이력 관리 + 사용량 제어 (실패 시 안전 회피) | 비서·초안 작성자 |
| **알림 디스패처 (Notification Dispatcher)** | 5채널 동시 발송 + AI 초안 본문 머지 | 우체국·집배원 |
| **PII 마스킹 필터 (PII Masking Filter)** | 입구 단계에서 PII(Personally Identifiable Information, 개인 식별 정보) 마스킹 (이메일·전화·긴 비밀값) | 검문소 |
| **DLQ 재처리 서비스 (DLQ Replay Service)** | 발송 실패 영속 보존 + 중복 없는 재발송 | 보험·블랙박스 |

**100% 검증**: 위 6개 구성 요소의 합 = AIOpsAgent 전체. 누락 영역 없음, 중복 영역 없음.

---

## 4. 품질 목표 (3대)

운영 약속으로서의 정량 지표 3개입니다. 이 3가지가 깨지면 AIOpsAgent는 신뢰받을 수 없습니다.

| 목표 | 의미 | 측정 기준 |
|---|---|---|
| **정보 손실 0** | 어떤 실패 상황에서도 운영 담당자는 알람과 SOP를 수신합니다. silent drop 없음. | drop 건수 0 |
| **30초 안 발송** | 장애 발생 → 메신저 도달까지 평균 30초 이내 (운영 담당자 승인 시간 제외) | p95 지연 ≤ 30초 |
| **100% 감사 기록** | 누가 언제 무엇을 수행했는지 모두 영속 기록 | 감사 누락률 0% |

세 목표는 한장 브리핑(brief.md)과 동일하며, 상세본 Overview의 QG-1·QG-2·QG-3과 정확히 일치합니다. AIOpsAgent가 이 세 목표를 보장합니다.

---

## 5. 통합 방식 결정 (ADR-001)

초기 시제품(PoC)은 Python 오케스트레이터(`ds_apm_poc`)와 SigNoz를 bridge로 연결하는 구조였습니다. 두 런타임(Python + Go)을 모두 모니터링·배포해야 했고, bridge가 재시도·DLQ·메신저 발송·SOP 저장을 중복 구현하는 문제가 발생했습니다. alert 페이로드 스키마가 두 시스템 간 미세하게 어긋나기 시작한 것도 주요 원인이었습니다.

**결정 (2026-05-19, ADR-001)**: Python 런타임을 폐기하고, AIOpsAgent의 모든 처리 단계를 SigNoz Community Go 코드라인에 내장된 확장 모듈로 구현합니다. SigNoz Alertmanager의 발송 핵심 경로(dispatcher hot path)에 SOP 그라운딩·AI 초안·DLQ 단계를 삽입하는 방식입니다.

**얻은 것**: Go 바이너리 단일 운영, 배포·롤백 표면 절반 축소, Alertmanager가 이미 보유한 5채널 발송 자산을 그대로 활용, 두 시스템 간 스키마 어긋남 해소.

**감수할 것**: SigNoz upstream의 내부 변경 영향, 6개 구성 요소를 Go로 재작성하는 공수.

---

## 6. 알려진 위험

착수 후 production-readiness까지 4개의 격차를 순차적으로 해소할 예정입니다.

| # | 위험 | 비즈니스 영향 |
|---|---|---|
| **R-1** | HMAC(Hash-based Message Authentication Code, 메시지 인증 코드) 서명 정책 미정 | 알림 재발송 시 위변조 검증 불가 — 보안 follow-up 필요 |
| **R-2** | 다중 테넌트 격리가 production-ready 아님 | 여러 고객사를 한 인스턴스로 운영할 경우 격리 강화 필요. 내부 단일 테넌트 운영에는 영향 없음 |
| **R-3** | 개인 식별 정보(PII) 마스킹이 입구 1점에만 적용 | OpenTelemetry Collector 단으로 이동 검토 권장 |
| **R-4** | 외부 LLM 의존 | **이미 해결됨** — LLM 실패 시 SOP 원문 그대로 전달(장애 시 통과(fail-open)). 정보 손실 0 보장 |

R-1·R-2·R-3은 §8 관련 산출물의 한장 브리핑(brief.html)에 의사결정 요청 사항(D-1·D-2·D-3)으로 함께 정리되어 있습니다.

---

## 7. 이해관계자별 책임

| 역할 | 책임 |
|---|---|
| **운영 담당자 (당직 운영자, on-call)** | AI 초안 검수·승인, 발송 결과 모니터링, 발송 실패 시 수동 재발송 |
| **SRE** | 시스템 이상 알람(AI 실패·DLQ 적재 급증) 수신 + 자격증명 회전 |
| **Platform Admin** | 테넌트 정책 등록, AI 전략 활성화, 감사 sink 설정 |
| **Security** | 개인 식별 정보 마스킹 정책·HMAC 정책·테넌트 격리 검토 |

---

## 8. 관련 산출물

| 산출물 | 누가 보나 | 링크 |
|---|---|---|
| **한장 브리핑** | 5분 안에 전체 상황 파악 | [brief.html](brief.html) |
| **요약본 Use Case** | 시나리오 3종 (정상·실패·AI 다운) | [brief-usecase.html](brief-usecase.html) |
| **요약본 기능명세** | 구성 요소 6개 요지 | [brief-spec.html](brief-spec.html) |
| **요약본 WBS** | 작업 진행 상황·마일스톤 | [brief-wbs.html](brief-wbs.html) |
| **상세본 Overview** | arc42 v9.0 전체 (아키텍처·품질 시나리오) | [../01-overview/index.md](../01-overview/index.md) |

본 개요는 상세본 Overview를 비기술 청중에 맞춰 압축한 것입니다. 상세 내용이 필요한 부분은 위 링크를 참조해 주십시오.

---

> 본 산출물은 개발 디테일 없이도 의사결정에 필요한 정보만 담았습니다.
> 추가 문의는 [한장 브리핑](brief.html) 또는 [상세본 Overview](../01-overview/index.md) 참조.
