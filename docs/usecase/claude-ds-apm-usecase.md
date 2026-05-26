# DS-APM 인시던트 핸드오프 — Use Case 다이어그램 (Mermaid)

> 출처: 실제 등록 라우트 (`pkg/apiserver/signozapiserver/ruler.go`) + 핸들러
> (`pkg/ruler/signozruler/*.go`) + AI/런북 드래프터 (`pkg/ruler/aigenerator`, `pkg/ruler/runbookdrafter`).
> 색상 규약 — 🟦 **View**(뷰어+운영자 가능) / 🟧 **Edit**(운영자만). 외부/시스템 액터는 회색.

---

## 1. 컨텍스트 다이어그램 (액터 ↔ 5개 유스케이스 그룹)

```mermaid
flowchart LR
    OP["👤 운영자 / SRE<br/><small>EditAccess</small>"]:::actor
    VW["👤 뷰어<br/><small>ViewAccess</small>"]:::actor
    LLM["🤖 LLM Provider<br/><small>Claude · Codex (API/CLI)</small>"]:::ext
    SRC["🌐 Pilot SOP 소스<br/><small>managed markdown</small>"]:::ext
    DISP["⚡ Alert Dispatch<br/><small>dispatchhook</small>"]:::ext

    subgraph SYS["DS-APM 인시던트 핸드오프 시스템"]
        direction TB
        G1(["Alert Rule 관리"])
        G2(["SOP 문서 관리"])
        G3(["Runbook 관리"])
        G4(["AI 전략 / 설정"])
        G5(["Downtime 스케줄"])
    end

    VW --- G1 & G2 & G3 & G4 & G5
    OP --- G1 & G2 & G3 & G4 & G5

    G3 -. «include» .-> LLM
    G4 -. «include» .-> LLM
    G2 -. «include» .-> SRC
    DISP -. «trigger» .-> G4

    classDef actor fill:#1e293b,color:#fff,stroke:#0f172a;
    classDef ext fill:#e2e8f0,color:#0f172a,stroke:#94a3b8,stroke-dasharray:4 3;
    style SYS fill:#f8fafc,stroke:#3b82f6,stroke-width:2px;
```

---

## 2. 상세 다이어그램 (전체 유스케이스)

> 운영자/SRE는 모든 케이스 수행 가능. 뷰어는 🟦 View 케이스만.
> 점선 화살표는 외부 액터에 대한 «include»/«trigger».

```mermaid
flowchart LR
    OP["👤 운영자 / SRE"]:::actor
    VW["👤 뷰어"]:::actor
    LLM["🤖 LLM Provider"]:::ext
    SRC["🌐 Pilot SOP 소스"]:::ext
    DISP["⚡ Alert Dispatch"]:::ext

    subgraph SYS["DS-APM 시스템"]
        direction TB

        subgraph S1["Alert Rule"]
            direction TB
            R1(["규칙 목록"]):::view
            R2(["규칙 조회"]):::view
            R3(["규칙 생성"]):::edit
            R4(["규칙 수정"]):::edit
            R5(["규칙 삭제"]):::edit
            R6(["규칙 패치"]):::edit
            R7(["규칙 테스트"]):::edit
            R8(["알림 템플릿 프리뷰"]):::edit
        end

        subgraph S2["SOP 문서"]
            direction TB
            P1(["SOP 문서 목록"]):::view
            P2(["SOP 문서 조회"]):::view
            P3(["SOP 버전 조회"]):::view
            P4(["SOP 바인딩 프리뷰"]):::view
            P5(["Pilot 소스 목록"]):::view
            P6(["Pilot 소스 헬스"]):::view
            P7(["SOP 문서 생성"]):::edit
            P8(["SOP 프리뷰"]):::edit
            P9(["Pilot managed markdown fetch"]):::edit
        end

        subgraph S3["Runbook"]
            direction TB
            B1(["런북 목록"]):::view
            B2(["런북 조회"]):::view
            B3(["런북 생성"]):::edit
            B4(["런북 수정"]):::edit
            B5(["런북 삭제"]):::edit
            B6(["런북 LLM 초안 생성"]):::edit
        end

        subgraph S4["AI 전략 / 설정"]
            direction TB
            A1(["AI 전략 프리뷰"]):::view
            A2(["최신 AI 전략 이력"]):::view
            A3(["AI 설정 조회"]):::view
            A4(["AI 설정 수정"]):::edit
            A5(["AI 설정 테스트"]):::edit
            A6(["AI 전략 자동 생성"]):::view
        end

        subgraph S5["Downtime 스케줄"]
            direction TB
            D1(["Downtime 목록"]):::view
            D2(["Downtime 조회"]):::view
            D3(["Downtime 생성"]):::edit
            D4(["Downtime 수정"]):::edit
            D5(["Downtime 삭제"]):::edit
        end
    end

    OP --- S1 & S2 & S3 & S4 & S5
    VW --- S1 & S2 & S3 & S4 & S5

    B6 -. «include» .-> LLM
    A1 -. «include» .-> LLM
    A5 -. «include» .-> LLM
    P9 -. «include» .-> SRC
    P6 -. «include» .-> SRC
    DISP -. «trigger» .-> A6

    classDef actor fill:#1e293b,color:#fff,stroke:#0f172a;
    classDef ext fill:#e2e8f0,color:#0f172a,stroke:#94a3b8,stroke-dasharray:4 3;
    classDef view fill:#dbeafe,color:#1e3a8a,stroke:#3b82f6;
    classDef edit fill:#ffedd5,color:#9a3412,stroke:#f97316;
    style SYS fill:#f8fafc,stroke:#64748b;
```

---

## 3. 유스케이스 ↔ 라우트 대응표

| 그룹 | 유스케이스 | Method · Path | 권한 |
|---|---|---|---|
| Alert Rule | 규칙 목록 | `GET /api/v2/rules` | View |
| Alert Rule | 규칙 조회 | `GET /api/v2/rules/{id}` | View |
| Alert Rule | 규칙 생성 | `POST /api/v2/rules` | Edit |
| Alert Rule | 규칙 수정 | `PUT /api/v2/rules/{id}` | Edit |
| Alert Rule | 규칙 삭제 | `DELETE /api/v2/rules/{id}` | Edit |
| Alert Rule | 규칙 패치 | `PATCH /api/v2/rules/{id}` | Edit |
| Alert Rule | 규칙 테스트 | `POST /api/v2/rules/test` | Edit |
| Alert Rule | 알림 템플릿 프리뷰 | `POST /api/v2/rules/notification_template/preview` | Edit |
| SOP 문서 | SOP 프리뷰 | `POST /api/v2/rules/sop/preview` | Edit |
| SOP 문서 | Pilot managed markdown fetch | `POST /api/v2/rules/sop/pilot/managed_markdown/fetch` | Edit |
| SOP 문서 | Pilot 소스 목록 | `GET /api/v2/ds/sop/sources` | View |
| SOP 문서 | Pilot 소스 헬스 | `GET /api/v2/ds/sop/sources/{id}/health` | View |
| SOP 문서 | SOP 문서 생성 | `POST /api/v2/ds/sop/documents` | Edit |
| SOP 문서 | SOP 문서 목록 | `GET /api/v2/ds/sop/documents` | View |
| SOP 문서 | SOP 문서 조회 | `GET /api/v2/ds/sop/documents/{sopId}` | View |
| SOP 문서 | SOP 버전 조회 | `GET /api/v2/ds/sop/documents/{sopId}/versions/{version}` | View |
| SOP 문서 | SOP 바인딩 프리뷰 | `POST /api/v2/ds/sop/bindings/preview` | View |
| Runbook | 런북 목록 | `GET .../{sopId}/versions/{version}/runbooks` | View |
| Runbook | 런북 조회 | `GET .../runbooks/{runbookId}` | View |
| Runbook | 런북 생성 | `POST .../runbooks` | Edit |
| Runbook | 런북 수정 | `PUT .../runbooks/{runbookId}` | Edit |
| Runbook | 런북 삭제 | `DELETE .../runbooks/{runbookId}` | Edit |
| Runbook | 런북 LLM 초안 생성 | `POST /api/v2/ds/runbooks/draft` | Edit |
| AI 전략/설정 | AI 전략 프리뷰 | `POST /api/v2/ds/ai/strategy/preview` | View |
| AI 전략/설정 | 최신 AI 전략 이력 | `GET /api/v2/ds/ai/strategy/history/latest` | View |
| AI 전략/설정 | AI 설정 조회 | `GET /api/v2/ds/ai/config` | View |
| AI 전략/설정 | AI 설정 수정 | `PUT /api/v2/ds/ai/config` | Edit |
| AI 전략/설정 | AI 설정 테스트 | `POST /api/v2/ds/ai/config/test` | Edit |
| AI 전략/설정 | AI 전략 자동 생성 | `dispatchhook` (알림 발생 시 내부 트리거) | — |
| Downtime | Downtime 목록 | `GET /api/v1/downtime_schedules` | View |
| Downtime | Downtime 조회 | `GET /api/v1/downtime_schedules/{id}` | View |
| Downtime | Downtime 생성 | `POST /api/v1/downtime_schedules` | Edit |
| Downtime | Downtime 수정 | `PUT /api/v1/downtime_schedules/{id}` | Edit |
| Downtime | Downtime 삭제 | `DELETE /api/v1/downtime_schedules/{id}` | Edit |
