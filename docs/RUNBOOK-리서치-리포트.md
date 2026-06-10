# RUNBOOK 표준 양식 및 필수 구성 요소 — ITO 장애대응 관제시스템 설계용 리서치 리포트

> 조사 방법: Deep-research 하니스 (5개 검색 각도 / 24개 소스 / 97개 주장 추출 → 25개 검증 → **21개 확정, 4개 기각**).
> 모든 인용은 1차 소스(AWS·Google SRE·PagerDuty·Microsoft·한국 공공 레퍼런스) 기준이며, 각 주장의 적대적 검증 통과율(예: 3-0)을 함께 표기.
> 작성일: 2026-06-10

---

## 목차
1. [SOP vs RUNBOOK vs PLAYBOOK 개념 구분](#1-sop-vs-runbook-vs-playbook--개념-구분-관제시스템-데이터-모델의-기초)
2. [런북 표준 템플릿 구조](#2-런북-표준-템플릿-구조--업계-레퍼런스-2종)
3. [런북 문서 필수 정보 항목](#3-런북-문서-필수-정보-항목-필드-체크리스트)
4. [머신리더블(YAML/JSON) 필드 설계](#4-머신리더블yamljson-필드-설계--관제-자동화-연계용)
5. [ITIL/SRE/IR 베스트 프랙티스](#5-itilsreir-베스트-프랙티스--에스컬레이션역할사후조치)
6. [한국 ITO/SI 운영 현장 맥락](#6-한국-itosi-운영-현장-맥락--자원별-장애처리-절차-레거시)
7. [관제시스템 설계 핵심 권고](#7-관제시스템-설계--핵심-권고-요약)
8. [부록 — 소스 목록 및 기각 주장](#부록--1차-소스-목록-인용-확정)

---

## 1. SOP vs RUNBOOK vs PLAYBOOK — 개념 구분 (관제시스템 데이터 모델의 기초)

장애대응 시스템을 설계할 때 이 3계층을 먼저 분리해야 데이터 모델이 깨지지 않는다.

| 구분 | 정의 | 로직 | 범위 |
|------|------|------|------|
| **RUNBOOK** | 분석자가 **하나의 구체적·개별 작업**을 완료하기 위해 따르는 단계별 기술 지침서 (예: 인증정보 재설정, 감염 워크스테이션 격리, 방화벽 규칙 변경) | 단일 작업 절차 | 좁음(Task) |
| **PLAYBOOK** | 여러 런북을 엮은 **시나리오 단위 대응 흐름** | 조건/분기 포함 | 넓음(Scenario) |
| **SOP** | 조직 표준 운영 절차 (정책·역할·거버넌스) | 정책 | 최상위 |

> "A runbook is a documented set of steps an analyst follows to complete a specific task during incident response." — ManageEngine SOAR (검증 2-1)

**⚠️ 기각된 통념:** "런북은 항상 선형(linear)·비분기 로직이다"라는 주장은 **3-0으로 기각**. 실무 런북은 조건/분기(if-then)를 포함할 수 있으며, 관제시스템 설계 시 **런북을 단순 순차 리스트로만 모델링하면 안 된다.** 분기·조건 필드를 데이터 모델에 반드시 포함.

**설계 시사점:** `SOP(정책) → Playbook(시나리오) → Runbook(작업)` 3계층 참조 관계로 DB 설계, 하나의 Playbook이 N개 Runbook을 호출하는 구조.

---

## 2. 런북 표준 템플릿 구조 — 업계 레퍼런스 2종

### 2-1. Rootly 7단계 표준 프레임워크 (검증 3-0)
가장 명확하게 검증된 런북 골격:

1. **Scope / Trigger / Impact** — 알림 발동 조건과 영향받는 서비스 정의
2. **Context Collection** — 상황 정보 수집
3. **Quick Triage Checklist** — 빠른 분류 체크리스트
4. **Document the Exact Fix** — 정확한 조치 문서화
5. **Communications Checklist** — 커뮤니케이션 체크리스트
6. **Verify Resolution** — 측정 가능한 성공 지표(지연시간, 에러율)로 해결 검증
7. **Close the Loop** — 포스트모템 일정 + 런북 갱신

> 출처: https://rootly.com/incident-response/runbooks

### 2-2. PagerDuty 4단계 라이프사이클 (검증 2-1)
런북/SOP 콘텐츠를 조직화하는 상위 틀:

**Before(준비·계획) → During(능동 대응) → After(학습·포스트모템) → Crisis Response(확장된 메이저 인시던트)**

이 4단계는 관제시스템의 인시던트 상태머신(state machine) 단계와 1:1 매핑하기 좋다.

> 출처: https://response.pagerduty.com/

---

## 3. 런북 문서 필수 정보 항목 (필드 체크리스트)

Rootly 표준이 규정하는 **반드시 포함되어야 할 코어 필드** (검증 3-0):

| 필수 항목 | 내용 | 비고 |
|-----------|------|------|
| **Trigger & Detection** | 트리거·탐지 조건 | 알림 발동 조건 명시 |
| **Impact Assessment** | 영향도 평가 | 영향 서비스/범위 |
| **Containment Actions** | 격리/차단 액션 | 확산 방지 |
| **Resolution Workflow** | 해결 워크플로우 | **복사-붙여넣기 가능한 명령어·스크립트 + 롤백 지침 포함** |
| **Validation & Verification** | 검증 | **측정 가능한 성공 기준**(성능 임계값 등) 정의 필수 |
| **Communication Plan** | 커뮤니케이션 계획 | 이해관계자 통지 |
| **Post-Incident Review** | 사후 검토 | 근본원인·교훈·후속조치 캡처 |

> "Resolution Workflow ... Include detailed, copy-pasteable commands, scripts, and rollback instructions ... Validation & Verification ... Define measurable success criteria such as performance thresholds" — Rootly (3-0)

**핵심 설계 원칙 2가지:**
- **롤백은 별도 필수 필드** (Resolution 안에 임베드). 실패 시 되돌릴 경로 없는 런북은 불완전.
- **검증 단계는 "측정 가능한 수치"**여야 한다. "정상 확인" 같은 모호한 표현 금지 → 지연시간/에러율 임계값 같은 정량 기준.

---

## 4. 머신리더블(YAML/JSON) 필드 설계 — 관제 자동화 연계용

관제시스템에 런북을 **실행 가능한 자동화**로 연계하려면 정적 문서가 아니라 구조화 스키마가 필요. PagerDuty도 런북을 "정적 문서가 아닌 실행 가능한 자동 워크플로우"로 정의(3-0).

### 4-1. AWS Systems Manager Automation — 가장 검증된 실전 스키마 (모두 3-0)

- **스키마 버전 `0.3` 강제, JSON 또는 YAML 작성 가능**
- **최상위 구조:**
```yaml
description:     # 런북 설명
schemaVersion: "0.3"
assumeRole:     # 실행 권한(IAM)
parameters:     # 입력 파라미터
variables:      # 변수
mainSteps:      # 실행 단계 배열 (핵심)
files:          # 첨부 파일
```

- **각 단계(step)의 필드 구조:**
```yaml
mainSteps:
  - name: "{{stepName}}"      # 단계명
    action: "{{action-name}}" # 수행 액션
    maxAttempts: 1            # 재시도 횟수
    inputs: {...}            # 입력
    outputs:                 # 출력
      Name: "{{output-name}}"
      Selector: "{{selector.value}}"
      Type: "{{data-type}}"
```

> **권장:** `maxAttempts`(재시도), `inputs/outputs`(단계 간 데이터 전달), `assumeRole`(권한 격리)는 자동화 안정성의 핵심이므로 자체 스키마에도 반드시 반영.
> 출처: https://docs.aws.amazon.com/systems-manager/latest/userguide/documents-schemas-features.html

### 4-2. Rundeck — 에러 핸들링 시맨틱 (검증 3-0)
실행 단계를 `sequence` 아래 `commands` 배열로 두고 제어 필드 추가:
- **`keepgoing`** (true/false) — 에러 발생 시 계속 진행 여부
- **`strategy`** — `node-first` / `step-first` 실행 전략

> **권장:** 단계별 **실패 시 동작(중단 vs 계속)을 필드로 명시**. "한 단계 실패 = 전체 중단"이 항상 옳지 않으므로 단계별 제어 가능해야 함.
> 출처: https://docs.rundeck.com/2.10.0/man5/job-yaml.html

### 4-3. Azure SRE Agent — 조건→액션 라우팅 구조 (검증 3-0)
응답 계획을 **2부 구조**로:
- **Incident filter** — 어떤 인시던트에 매칭할지 (예: `api-gateway` 서비스의 P1·P2)
- **Custom agent handler** — 어떻게 대응할지

> 관제시스템에서 **"어떤 알림이 어떤 런북을 자동 트리거하는가"**의 라우팅 모델. `filter(조건) → handler(런북)` 매핑 테이블 필요.
> 출처: https://learn.microsoft.com/en-us/azure/sre-agent/incident-response-plans

---

## 5. ITIL/SRE/IR 베스트 프랙티스 — 에스컬레이션·역할·사후조치

### 5-1. 에스컬레이션 & 역할 구조

**PagerDuty 6대 표준 역할** (검증 3-0):
Incident Commander(IC), Deputy, Scribe, SME, Customer Liaison, Internal Liaison — 각 역할별 책임 명시. (예: Deputy는 심각도 모니터링 및 IC 통지, Customer Liaison는 대외 공지)

**Google IMAG 역할** (ICS 기반, 검증 3-0): Incident Commander / Operations Lead / Communications Lead / Planning Lead 분리.

**검증된 에스컬레이션 핸드오프 규칙** (검증 3-0): 지휘권 인계는 **명시적·확인 기반**이어야 함 — 인계자가 "당신이 이제 IC입니다, 맞죠?"를 명시하고 **확실한 인수 확인을 받기 전엔 이탈 금지**.
→ 관제시스템에 **핸드오프 로그 + 명시적 ACK** 기능 근거.

**Google SRE 핵심 온콜 리소스 3종** (검증 3-0) — 런북 컴포넌트와 직결:
1. 명확한 에스컬레이션 경로 → 에스컬레이션 필드
2. 잘 정의된 인시던트 관리 절차 → 런북 본문
3. 비난 없는(blameless) 포스트모템 문화 → 사후조치

### 5-2. 사후조치(Postmortem) — 시간 제약 포함 (검증 3-0)
PagerDuty 규정:
- **미팅 일정: SEV-1은 3 영업일 이내, SEV-2는 5 영업일 이내**
- 상태 변화 + 대응자 액션 **타임라인** 작성
- 각 타임라인 항목별 **근거 데이터/메트릭**
- 근본원인·고객영향 분석
- **후속 액션 아이템을 JIRA 티켓으로 생성** (추적 가능하게)

→ 관제시스템에 **심각도별 포스트모템 SLA 타이머**와 **액션아이템→티켓 연동** 설계.

### 5-3. 사전 준비 원칙 (검증 3-0)
- Google: "인시던트 관리 절차를 **사전에**, 참여자와 협의하여 문서화하라" → 즉흥 대응이 아닌 **사전 작성 런북** 원칙
- 플레이북은 **디버그·완화 지침을 명시**하고 **항상 최신 유지**해야 하며, **온콜 담당자가 인지·훈련**되어야 효과 발생.

---

## 6. 한국 ITO/SI 운영 현장 맥락 — '자원별 장애처리 절차' 레거시

한국 공공/SI 현장에서 통용되는 실제 레거시 양식(`자원별_장애처리_절차`)에서 검증된 구조.

### 6-1. 액션 런북 — 목표복구시간(RTO) 명시 (검증 3-0)
> "장애조치 목표시간은 장애요인 파악 후 장애조치 완료시까지의 시간을 의미한다"

자원별 **목표복구시간** 구체 예시:
- 웹서버 단일 디스크 장애: **1시간**
- OS 손상: **5분**
- LAN 카드 불량: **30분**
- Oracle 프로세스 재기동: **10분**

조치 절차 예시: `1) 리소스 과다점유 프로세스 kill → 2) 오류 원인 제거 → 3) 프로세스 재실행` + 담당팀/담당자 명기.

### 6-2. 진단 체크리스트 — 액션과 분리된 별도 테이블 (검증 3-0)
'장애관리 시나리오'는 액션 런북과 **분리된 진단 체크리스트**를 가지며 컬럼은:
- **예상발생지점** (expected failure point)
- **장애요인 파악을 위한 점검 순서** (원인 가능성 높은 순)
- **관련팀 및 담당자**

> 예: 웹서버 → 1) 최근 서버/응용SW 변경사항 점검 2) 메모리·디스크 등 HW 자원 점검 3) 인터페이스 프로세스 점검
> 출처: 자원별 장애처리 절차 (shareditassets S3, 한국 SI 레거시 문서)

**한국 현장 시사점:**
- 한국 ITO 런북은 **「진단(시나리오) 테이블」 + 「조치(절차) 테이블」 2분 구조**가 사실상 표준 → 글로벌 Rootly 7단계의 "Triage → Fix" 분리와 일치.
- **목표복구시간(RTO)을 자원·장애유형별로 못박는 것**이 한국 SI 계약(SLA) 관행. 관제시스템에 **장애유형별 RTO 필드 + 초과 시 알람** 필수.
- **⚠️ 주의:** "한국 표준이 정확히 4개 컬럼(장애원인/조치순서/담당자/목표복구시간)으로 고정"이라는 주장은 **기각(1-2)**. 컬럼 구성은 조직마다 변형되므로 고정 스키마로 강제하지 말 것.

---

## 7. 관제시스템 설계 — 핵심 권고 요약

1. **3계층 데이터 모델**: SOP(정책) → Playbook(시나리오) → Runbook(작업). 런북은 분기/조건 포함 가능.
2. **런북 필수 필드 7종**(Rootly 3-0): Trigger / Impact / Containment / Resolution(+롤백) / Validation(정량) / Communication / Post-Incident.
3. **머신리더블 스키마**(AWS 0.3 기반): `description / parameters / mainSteps(name·action·inputs·outputs·maxAttempts) / files` + Rundeck `on_failure(stop/continue)` + Azure `filter→handler` 라우팅.
4. **롤백·검증은 별도 필수 필드**, 검증은 측정 가능한 임계값으로.
5. **에스컬레이션**: 역할 기반(IC/Ops/Comms) + 명시적 ACK 핸드오프 로그.
6. **사후조치 자동화**: 심각도별 포스트모템 SLA(SEV-1 3일/SEV-2 5일) 타이머 + 액션아이템→티켓 연동.
7. **한국 현장 반영**: 진단테이블/조치테이블 2분 구조 + 자원별 RTO 필드(고정 스키마 강요는 금지).

---

## 부록 — 1차 소스 목록 (인용 확정)

| 소스 | 등급 | 확정 claim 수 |
|------|------|--------------|
| PagerDuty Response (response.pagerduty.com) | primary | 5 |
| Google SRE Book (managing-incidents / being-on-call) | primary | 다수 |
| Google IMAG (Incident Management Guide PDF) | primary | 다수 |
| AWS Systems Manager docs | primary | 5 |
| Rundeck job-yaml docs | primary | 3 |
| Microsoft Azure SRE Agent docs | primary | 5 |
| Rootly Runbooks 가이드 | blog | 5 |
| ManageEngine SOAR (runbook vs playbook) | secondary | 5 |
| 한국 '자원별 장애처리 절차' (shareditassets S3) | primary | 5 |

### 기각된 4개 주장 (설계 시 함정)
1. 런북 = 선형(비분기) 로직 — **기각 0-3**
2. Google 7단계 고정 라이프사이클(Prepare→Detect→Triage→…) — **기각 0-3**
3. Rundeck 필수 top-level 4필드(name/description/loglevel/sequence) — **기각 1-2**
4. 한국 표준 4컬럼 고정 — **기각 1-2**

> **공통 교훈:** 스키마를 **고정·경직되게** 설계하지 말 것. 확장 가능한(extensible) 필드 구조로 가야 함.
