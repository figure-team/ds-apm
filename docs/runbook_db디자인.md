# 런북 기반 장애대응 관제시스템 — DB 테이블 설계서

> 리서치 리포트 및 `runbook.schema.json` 기반 RDB 스키마 설계.
> 3계층 모델(SOP → Playbook → Runbook) + 실행 인스턴스(Incident) 분리.
> 대상 DBMS: PostgreSQL 기준 (Oracle/MySQL 이식 가능). 작성일: 2026-06-10

---

## 1. 설계 원칙

1. **정의(Definition)와 실행(Execution) 분리** — 런북 "정의"(템플릿)와 실제 장애 발생 시 "실행 인스턴스"를 다른 테이블로 분리. 정의는 버전 관리, 실행은 감사 로그.
2. **3계층 참조** — `sop` → `playbook` → `runbook` → `runbook_step`.
3. **유연한 스키마** — 검증에서 기각된 "고정 컬럼 강제" 함정을 피하기 위해, 가변 항목은 정규화 + `JSONB` 확장 필드 병행.
4. **한국 SI 맥락 반영** — `rto_minutes`(목표복구시간), 진단 체크리스트(`diagnosis_check`) 별도 테이블.
5. **머신리더블 자동화** — `runbook_step`에 `action_type`, `command`, `on_failure`, 분기 필드 포함.

---

## 2. ERD (논리 모델)

```
┌─────────┐      ┌──────────┐      ┌──────────┐      ┌────────────────┐
│   sop   │1────*│ playbook │1────*│ runbook  │1────*│  runbook_step  │
└─────────┘      └──────────┘      └────┬─────┘      └────────────────┘
                                        │1
                        ┌───────────────┼───────────────┬──────────────┐
                        │*              │*              │*             │*
               ┌────────────────┐ ┌──────────┐ ┌────────────────┐ ┌──────────────┐
               │ runbook_trigger│ │ rollback │ │ diagnosis_check│ │ verification │
               └────────────────┘ │  _step   │ └────────────────┘ │  _criterion  │
                                   └──────────┘                    └──────────────┘
            ┌──────────────────┐
   runbook *│ escalation_level │* (소속: escalation_policy)
            └──────────────────┘

  ┌─────────────────────────── 실행(Execution) 영역 ───────────────────────────┐
  │  ┌──────────┐      ┌────────────────────┐      ┌─────────────────────────┐ │
  │  │ incident │1────*│ incident_step_log  │      │ incident_timeline_event │ │
  │  └────┬─────┘      └────────────────────┘      └─────────────────────────┘ │
  │       │1                                                                    │
  │       │*                                                                    │
  │  ┌──────────────────┐   ┌──────────────────┐                               │
  │  │ postmortem        │1─*│ postmortem_action│                               │
  │  └──────────────────┘   └──────────────────┘                               │
  └────────────────────────────────────────────────────────────────────────────┘

  incident.runbook_id ───▶ runbook.id  (어떤 런북으로 대응했는지)
  incident.escalation_policy_id ─▶ escalation_policy
```

---

## 3. 테이블 정의 (DDL)

### 3-1. 정의(Definition) 영역

#### `sop` — 표준 운영 절차 (최상위 정책)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| code | VARCHAR(50) | UNIQUE, NOT NULL | SOP 코드 |
| title | VARCHAR(255) | NOT NULL | 제목 |
| description | TEXT | | 개요 |
| owner_team | VARCHAR(100) | | 소유 팀 |
| created_at | TIMESTAMPTZ | DEFAULT now() | |
| updated_at | TIMESTAMPTZ | | |

#### `playbook` — 시나리오 단위 대응 흐름
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| sop_id | BIGINT | FK→sop.id | 소속 SOP |
| code | VARCHAR(50) | UNIQUE, NOT NULL | |
| title | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| created_at | TIMESTAMPTZ | DEFAULT now() | |

#### `runbook` — 개별 작업 런북 (핵심 정의 테이블)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| playbook_id | BIGINT | FK→playbook.id, NULL 허용 | 소속 플레이북(독립 런북 가능) |
| code | VARCHAR(50) | NOT NULL | 런북 코드 (예: RB-WEB-DISK-001) |
| version | VARCHAR(20) | NOT NULL | 시맨틱 버전 |
| title | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| category | VARCHAR(50) | | 자원 유형(WEB/WAS/DB/NETWORK/OS…) |
| severity | VARCHAR(10) | | SEV-1~4 / P1~4 |
| owner_team | VARCHAR(100) | | 담당 팀 |
| owner_person | VARCHAR(100) | | 담당자 |
| rto_minutes | INT | | **목표복구시간(분)** — 한국 SLA |
| service | JSONB | | 영향 서비스 배열 |
| impact_assessment | TEXT | | 영향도 평가 |
| status | VARCHAR(20) | DEFAULT 'active' | active/deprecated/draft |
| review_cycle_days | INT | DEFAULT 90 | 재검토 주기 |
| last_updated | DATE | | 최종 갱신일 |
| extra | JSONB | | 확장 필드(고정스키마 회피) |
| created_at | TIMESTAMPTZ | DEFAULT now() | |
| | | UNIQUE(code, version) | 코드+버전 유니크 |

#### `runbook_trigger` — 트리거/탐지 조건 (1 런북 : N 조건)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| runbook_id | BIGINT | FK→runbook.id, NOT NULL | |
| source | VARCHAR(100) | | 알림 소스(Prometheus/Zabbix…) |
| metric | VARCHAR(100) | NOT NULL | 지표명 |
| operator | VARCHAR(10) | NOT NULL | >, >=, <, ==, contains… |
| threshold | VARCHAR(100) | NOT NULL | 임계값 |
| duration | VARCHAR(20) | | 지속시간(예: 5m) |
| match_service_pattern | VARCHAR(255) | | 자동 라우팅 서비스 패턴 |
| match_severity | JSONB | | 매칭 심각도 배열 |

> **자동 라우팅:** 관제 알람 수신 시 `runbook_trigger`를 매칭해 해당 런북 자동 추천/실행 (Azure SRE filter→handler 모델).

#### `runbook_step` — 단계별 대응 액션 (핵심)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| runbook_id | BIGINT | FK→runbook.id, NOT NULL | |
| step_order | INT | NOT NULL | 실행 순서 |
| name | VARCHAR(255) | NOT NULL | 단계명 |
| action_type | VARCHAR(20) | NOT NULL | manual/automated/semi-automated |
| instruction | TEXT | NOT NULL | 사람이 읽는 지침 |
| command | TEXT | | **복붙 가능한 명령/스크립트** |
| inputs | JSONB | | 입력값 |
| outputs | JSONB | | 출력(다음 단계 전달) |
| max_attempts | INT | DEFAULT 1 | 재시도 |
| timeout_seconds | INT | | 타임아웃 |
| on_failure | VARCHAR(20) | DEFAULT 'stop' | stop/continue/escalate/goto_rollback |
| condition | TEXT | | **조건부 실행 표현식(분기)** |
| next_step_on_success | VARCHAR(255) | | 성공 시 분기 단계명 |
| next_step_on_failure | VARCHAR(255) | | 실패 시 분기 단계명 |
| | | UNIQUE(runbook_id, step_order) | |

#### `rollback_step` — 롤백 절차 (별도 필수)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| runbook_id | BIGINT | FK→runbook.id, NOT NULL | |
| step_order | INT | NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| command | TEXT | | |
| instruction | TEXT | | |

#### `diagnosis_check` — 진단 체크리스트 (한국 '장애관리 시나리오')
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| runbook_id | BIGINT | FK→runbook.id, NOT NULL | |
| check_order | INT | NOT NULL | 점검 순서(원인 가능성 높은 순) |
| expected_failure_point | VARCHAR(255) | NOT NULL | 예상발생지점 |
| check_action | TEXT | NOT NULL | 점검 항목 |
| related_team | VARCHAR(100) | | 관련팀 |
| owner | VARCHAR(100) | | 담당자 |

#### `verification_criterion` — 검증 기준 (정량)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| runbook_id | BIGINT | FK→runbook.id, NOT NULL | |
| metric | VARCHAR(100) | NOT NULL | 검증 지표 |
| operator | VARCHAR(10) | NOT NULL | |
| threshold | VARCHAR(100) | NOT NULL | 측정 가능한 기준값 |
| description | TEXT | | |

#### `escalation_policy` / `escalation_level` — 에스컬레이션 경로
**escalation_policy**
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| name | VARCHAR(100) | UNIQUE, NOT NULL | 정책명 |
| handoff_requires_ack | BOOLEAN | DEFAULT true | 명시적 ACK 필수 |

**escalation_level**
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| policy_id | BIGINT | FK→escalation_policy.id | |
| level | INT | NOT NULL | 단계 |
| trigger_after_minutes | INT | | 경과 시 에스컬레이션 |
| role | VARCHAR(50) | | IC/Deputy/Ops Lead/Comms Lead/SME |
| contact_team | VARCHAR(100) | | |
| contact_person | VARCHAR(100) | | |
| channel | VARCHAR(50) | | 전화/SMS/Slack |

> `runbook`은 `escalation_policy_id`(FK, NULL 허용)를 가져 정책을 참조하거나, 인시던트 단위로 지정.

---

### 3-2. 실행(Execution) 영역 — 감사·추적

#### `incident` — 장애 실행 인스턴스
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| incident_no | VARCHAR(50) | UNIQUE, NOT NULL | 장애번호 |
| runbook_id | BIGINT | FK→runbook.id | 적용된 런북 |
| runbook_version | VARCHAR(20) | | 적용 시점 런북 버전(스냅샷) |
| severity | VARCHAR(10) | | |
| status | VARCHAR(20) | | detected/triaging/mitigating/resolved/closed |
| service | VARCHAR(255) | | 영향 서비스 |
| detected_at | TIMESTAMPTZ | NOT NULL | 발생/탐지 시각 |
| resolved_at | TIMESTAMPTZ | | 해결 시각 |
| rto_target_minutes | INT | | 목표복구시간 |
| rto_breached | BOOLEAN | DEFAULT false | **RTO 초과 여부(알람 트리거)** |
| incident_commander | VARCHAR(100) | | IC |
| escalation_policy_id | BIGINT | FK→escalation_policy.id | |
| root_cause | TEXT | | 근본원인 |
| created_at | TIMESTAMPTZ | DEFAULT now() | |

#### `incident_step_log` — 단계 실행 로그
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| incident_id | BIGINT | FK→incident.id, NOT NULL | |
| runbook_step_id | BIGINT | FK→runbook_step.id | 실행 단계 |
| step_order | INT | | |
| status | VARCHAR(20) | | pending/running/success/failed/skipped |
| executed_by | VARCHAR(100) | | 수행자(또는 system) |
| attempts | INT | DEFAULT 0 | 시도 횟수 |
| output | JSONB | | 실행 결과 |
| started_at | TIMESTAMPTZ | | |
| finished_at | TIMESTAMPTZ | | |

#### `incident_timeline_event` — 타임라인 (포스트모템 자료)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| incident_id | BIGINT | FK→incident.id, NOT NULL | |
| event_at | TIMESTAMPTZ | NOT NULL | 시각 |
| event_type | VARCHAR(30) | | status_change/action/escalation/handoff/comm |
| actor | VARCHAR(100) | | 행위자 |
| description | TEXT | | |
| metric_snapshot | JSONB | | 근거 데이터/메트릭 |

#### `postmortem` — 사후 검토
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| incident_id | BIGINT | FK→incident.id, UNIQUE | |
| meeting_sla_due | DATE | | SLA 기한(SEV-1=3일/SEV-2=5일) |
| meeting_held_at | TIMESTAMPTZ | | 실제 미팅일 |
| root_cause | TEXT | | |
| customer_impact | TEXT | | |
| lessons_learned | TEXT | | |
| is_blameless | BOOLEAN | DEFAULT true | |

#### `postmortem_action` — 후속 액션 아이템 (티켓 연동)
| 컬럼 | 타입 | 제약 | 설명 |
|------|------|------|------|
| id | BIGSERIAL | PK | |
| postmortem_id | BIGINT | FK→postmortem.id, NOT NULL | |
| title | VARCHAR(255) | NOT NULL | |
| owner | VARCHAR(100) | | |
| due_date | DATE | | |
| ticket_system | VARCHAR(50) | | JIRA 등 |
| ticket_id | VARCHAR(100) | | 외부 티켓 ID |
| status | VARCHAR(20) | DEFAULT 'open' | open/in_progress/done |

---

## 4. 핵심 인덱스 권고

```sql
-- 자동 라우팅: 알람 → 런북 매칭 조회
CREATE INDEX idx_trigger_metric ON runbook_trigger (metric);
CREATE INDEX idx_runbook_category_severity ON runbook (category, severity);

-- 런북 단계 정렬 조회
CREATE INDEX idx_step_runbook_order ON runbook_step (runbook_id, step_order);

-- 실행 추적/대시보드
CREATE INDEX idx_incident_status ON incident (status);
CREATE INDEX idx_incident_detected ON incident (detected_at DESC);
CREATE INDEX idx_incident_rto_breached ON incident (rto_breached) WHERE rto_breached = true;
CREATE INDEX idx_steplog_incident ON incident_step_log (incident_id);

-- 포스트모템 SLA 추적
CREATE INDEX idx_postmortem_sla ON postmortem (meeting_sla_due) WHERE meeting_held_at IS NULL;

-- JSONB 검색 (서비스/확장 필드)
CREATE INDEX idx_runbook_service_gin ON runbook USING GIN (service);
CREATE INDEX idx_runbook_extra_gin ON runbook USING GIN (extra);
```

---

## 5. 정규화 vs JSONB 판단 기준

| 데이터 | 저장 방식 | 이유 |
|--------|-----------|------|
| 단계/진단/검증/에스컬레이션 | **정규화 테이블** | 순서·조회·자동실행에 구조 필요 |
| 영향 서비스 목록, 태그 | **JSONB** | 가변 배열, 검색 위주 |
| `runbook.extra` | **JSONB** | 조직별 커스텀 필드(고정스키마 강요 회피) |
| 단계 inputs/outputs | **JSONB** | 자동화 페이로드, 스키마 가변 |

---

## 6. 운영 쿼리 예시

```sql
-- (1) 디스크 사용률 알람 → 매칭 런북 자동 추천
SELECT r.id, r.code, r.title, r.rto_minutes
FROM runbook r
JOIN runbook_trigger t ON t.runbook_id = r.id
WHERE t.metric = 'disk_usage_pct'
  AND r.status = 'active'
ORDER BY r.severity;

-- (2) RTO 초과 진행 중 장애 (관제 대시보드 알람)
SELECT incident_no, service, severity,
       EXTRACT(EPOCH FROM (now() - detected_at))/60 AS elapsed_min,
       rto_target_minutes
FROM incident
WHERE status NOT IN ('resolved','closed')
  AND EXTRACT(EPOCH FROM (now() - detected_at))/60 > rto_target_minutes;

-- (3) 포스트모템 SLA 임박/초과 (SEV-1 3일/SEV-2 5일)
SELECT i.incident_no, p.meeting_sla_due
FROM postmortem p
JOIN incident i ON i.id = p.incident_id
WHERE p.meeting_held_at IS NULL
  AND p.meeting_sla_due <= CURRENT_DATE + INTERVAL '1 day';
```

---

## 7. 설계 요약

- **정의/실행 분리**로 런북 템플릿은 버전 관리, 장애 실행은 완전 감사 로그 확보.
- **자동화 연계**: `runbook_trigger`(라우팅) + `runbook_step`(action_type/command/on_failure/분기)로 머신리더블 실행 엔진 연결.
- **한국 SI 반영**: `rto_minutes` + `rto_breached` 알람, `diagnosis_check` 진단 테이블 분리.
- **유연성**: 가변 항목은 JSONB(`extra`)로 — 검증에서 기각된 "고정 컬럼 강제" 함정 회피.
- **사후조치 자동화**: `postmortem.meeting_sla_due` + `postmortem_action.ticket_id`로 SLA 타이머·티켓 연동.
