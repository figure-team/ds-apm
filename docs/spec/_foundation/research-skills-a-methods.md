# 01-A. 산출물별 작성 방법론 리서치

> Agent A 결과물 (분할). 이후 `research-skills.md`로 통합 예정.
> 작성일: 2026-05-28
> 대상: DS-APM (SigNoz 위에 얹은 "Incident → SOP runbook → Operator handoff" Go 확장 레이어)
> 범위: Overview / Use Case / 기능명세서(SRS) / WBS 4종을 HTML로 출력하기 위한 작성 표준

---

## 0. 공통 원칙 (HTML 산출물에 적용)

산출물 4종 전체에 공통으로 적용되는 HTML-네이티브 원칙. Pretext-스타일 HTML 문서가 마크다운보다 효과적인 이유는 (1) 공간 정보(diff, 콜그래프, 시퀀스)가 평탄화되지 않고, (2) 인터랙션이 묘사가 아니라 시연되며, (3) side-by-side 비교가 가능하기 때문이다.

- **공간 정보 보존**: 다이어그램, 콜그래프, 타임라인은 마크다운에 풀어쓰지 말고 SVG/Mermaid/구조화 레이아웃으로 렌더링
- **스캐너블 구조**: 헤더-색-여백을 활용해 "읽는 문서"가 아니라 "스캔하는 문서"로 설계
- **자가완결 단일 파일**: 브라우저 더블클릭으로 열리는 standalone HTML — 빌드 단계 없음
- **목적별 포맷**: 타임라인은 간트, 프로세스는 플로우, 컴포넌트는 컨택트시트

(참조: <https://thariqs.github.io/html-effectiveness/>)

---

## 1. Overview — 권장: **arc42 + C4 Context/Container 레벨 하이브리드**

### 1.1 권장 방법론

| 후보 | 추천 |
|---|---|
| **arc42** (전체 구조) | **★ 채택** |
| **C4 model** (다이어그램만 차용) | **★ 보조 채택 — Lv1/Lv2** |
| Diátaxis | 부분 차용 (Explanation 톤) |
| GitHub README anatomy | 최상단 헤더/배지 차용 |

**택일 추천 → arc42를 메인 골격으로, C4의 Context/Container 다이어그램을 5절(Building Block View)에 임베드.**

### 1.2 채택 근거

- DS-APM은 "SigNoz 위에 얹은 확장 레이어"라는 **외부 시스템과의 명확한 경계 정의**가 핵심이다. arc42 §3 (Context & Scope)와 C4 System Context 다이어그램은 정확히 이 문제를 풀기 위한 도구다.
- arc42는 12개 섹션이 모두 **선택적(optional)**이라는 점이 중요하다 — POC 단계 산출물에서 §11(Risks) §12(Glossary)만 채우고 나머지를 비워도 표준을 위반하지 않는다.
- arc42는 **CC BY-SA 4.0 무료 공개**, C4는 notation/tooling independent. 라이선스 리스크 zero.
- 한국 SI/대기업 산출물 문화에 "12개 섹션 골격"은 친숙(폭포수 SRS 익숙한 PM/PL에게 낯설지 않음)하면서도, 각 섹션이 옵셔널이라 스타트업 톤도 가능하다.
- Diátaxis는 **사용자 문서**용 프레임워크이지 아키텍처 문서용이 아니다 → Overview에는 부적합. 단, "이 문서는 Explanation 카테고리"라는 메타 인식만 차용.

### 1.3 표준이 정의하는 섹션 구조

**arc42 v9.0 (2025년 7월 기준) 12 섹션:**

1. **Introduction & Goals** — 요구사항 핵심, 품질 목표 top 3-5, 이해관계자 표
2. **Constraints** — 기술적/조직적/관습적 제약
3. **Context & Scope** — 시스템 경계, 외부 인터페이스 (비즈니스 컨텍스트 + 기술 컨텍스트)
4. **Solution Strategy** — 핵심 결정과 접근방식 요약
5. **Building Block View** — 정적 분해 (Level 1 = white box of system + black boxes of children, Level 2/3로 zoom-in)
6. **Runtime View** — 실행 시 컴포넌트 간 상호작용 시나리오
7. **Deployment View** — 인프라/환경/하드웨어 매핑
8. **Crosscutting Concepts** — 여러 부분에 걸친 원칙(로깅, 보안, i18n 등)
9. **Architectural Decisions** — ADR 형태로 중요한 결정의 근거
10. **Quality Requirements** — 10.1 Quality Tree + 10.2 Quality Scenarios (v9.0에서 재구조화)
11. **Risks & Technical Debt** — 알려진 위험/부채
12. **Glossary** — 용어집

**C4 4레벨 (보조):**

- **Level 1 — System Context**: DS-APM과 외부 시스템(SigNoz, 메신저 플랫폼, Operator, Incident source)
- **Level 2 — Container**: 배포 가능 단위 (Go 서비스, ClickHouse, Redis 등)
- **Level 3 — Component**: 컨테이너 내부 컴포넌트
- **Level 4 — Code**: 보통 생략(자동생성 가능). 보조 다이어그램으로 **System Landscape**, **Dynamic**, **Deployment**도 제공.

### 1.4 DS-APM 적용 시 주의점·조정 사항

- **§3 Context는 양분**: "비즈니스 컨텍스트(Incident → SOP → Operator)" + "기술 컨텍스트(SigNoz collector → DS-APM → 메신저 webhook)" 두 다이어그램을 같이 둔다. SigNoz와의 경계가 가장 중요한 그림이므로 페이지 폴드 안에 배치.
- **§5 Building Block은 Lv1만**: POC 단계에서 Lv2 white box를 깊게 그리면 유지비용 폭증. C4 Container 다이어그램을 Lv1로 사용하고, 핵심 컴포넌트 1-2개만 Lv2로 zoom-in.
- **§9 ADR**: "왜 Python ds_apm_poc를 폐기하고 Go + SigNoz로 갔는가" (memory의 마이그레이션 결정)를 ADR-001로 기록 → 한국 SI에서 "왜 이렇게 결정했냐"고 묻는 감리/리뷰 대응에 강력.
- **§10 Quality Requirements**: SigNoz 위에 얹는 레이어이므로 "지연(p99 < Xms)", "메신저 전달 실패율", "SOP 정합성"이 핵심 시나리오.
- **§11 Risks**: "var/signoz는 우리 코드" 같은 nested repo 위험을 명시.
- arc42를 **그대로 12장 다 쓰지 말고**, POC 단계는 §1, §3, §5, §6, §10, §11, §12 정도로 압축. "옵셔널이다"는 점이 표준의 강점.

### 1.5 HTML 렌더링 시 유리한 시각화 포인트

- **§3 Context 다이어그램**: SVG로 SigNoz/메신저/Operator를 박스로, DS-APM을 중앙 하이라이트. 호버 시 인터페이스 명세 툴팁.
- **§5 Building Block**: 클릭 시 Lv1→Lv2 드릴다운 인터랙션. C4 스타일의 nested box.
- **§6 Runtime View**: 시퀀스 다이어그램(Mermaid `sequenceDiagram`)으로 "Incident 수신 → SOP 매칭 → 메신저 전달" 흐름.
- **§9 ADR**: 카드 그리드 레이아웃 — 각 ADR이 카드 한 장(상태/날짜/제목/근거).
- **§10 Quality Tree**: 트리 형태 SVG, 각 leaf는 측정 가능한 시나리오.
- **§12 Glossary**: alphabet jump nav + 검색창.

### 1.6 참고 링크

- [arc42 Overview (공식)](https://arc42.org/overview)
- [arc42 Documentation (full template)](https://docs.arc42.org/home/)
- [arc42 §5 Building Block View](https://docs.arc42.org/section-5/)
- [arc42 Building Block Examples](https://docs.arc42.org/examples/)
- [C4 model 공식 홈](https://c4model.com/)
- [Diátaxis (참고용)](https://diataxis.fr/)
- [Make a README (최상단 헤더 차용)](https://www.makeareadme.com/)

---

## 2. Use Case — 권장: **Cockburn 표준 템플릿 + Gherkin (에러 케이스만)**

### 2.1 권장 방법론

| 후보 | 추천 |
|---|---|
| **Cockburn use case template** (Fully Dressed) | **★ 채택 — 메인 골격** |
| **Gherkin/BDD** | **★ 보조 채택 — 에러 케이스 상세화 전용** |
| UML use case diagram (OMG) | 보조 — 액터-유스케이스 관계도용 |
| Event Storming (Big Picture) | 차용 — Use Case 도출 단계의 사고도구 |
| UML Sequence Diagram + combined fragments (alt/opt/break) | **★ 채택 — 에러 시퀀스 시각화** |

**택일 추천 → Cockburn Fully Dressed 템플릿을 기본형으로, "SOP→메신저 전달 에러 케이스 1~2건"은 Gherkin Scenario로 추가 상세화하고, 시퀀스는 UML alt/break 프래그먼트로 그린다.**

### 2.2 채택 근거

- Cockburn 템플릿은 1990년대 초부터 **사실상 산업 표준**이고, 한국 SI/대기업 PM/QA가 가장 익숙한 형식이다. "정상 흐름 + 확장(Extension)"이라는 구조 자체가 "SOP→메신저 정상 전달 + 전달 실패 케이스"라는 우리 도메인과 완벽히 매핑된다.
- Cockburn의 **Extension 번호 규칙(3a, 3a1)** 은 에러 케이스를 메인 시나리오 단계에 정확히 앵커링한다 → "어느 스텝에서 실패했는지"가 명시적.
- Gherkin은 **실행 가능한 명세**다. SOP→메신저 전달 에러 케이스를 Gherkin으로 쓰면 Cucumber/godog 같은 도구로 회귀 테스트화 가능. 한국 스타트업 QA 톤에도 자연스럽다.
- UML 시퀀스의 `alt`, `opt`, `break` 프래그먼트는 **에러 분기를 한 다이어그램에 표현**할 수 있어 메신저 전달 실패 시각화에 적합. `break`는 특히 "예외 발생 시 enclosing fragment의 나머지를 건너뛴다"는 의미라 에러 패스에 정확히 매칭.
- Event Storming의 Big Picture는 **Use Case 도출 단계**(워크숍)에서 쓰는 사고도구로 차용. 최종 산출물 포맷은 아니다.

### 2.3 표준이 정의하는 섹션 구조

**Cockburn Fully Dressed Use Case 필수 필드:**

- **Use Case Name** (액티브 동사구, 예: "Incident 수신하고 SOP runbook 전달하기")
- **Goal in Context** — 1-2줄 요약
- **Scope** — System / Subsystem / Enterprise
- **Level** — Summary (Cloud) / User-goal (Sea) / Subfunction (Fish)
- **Primary Actor** — 누가 트리거하는가
- **Stakeholders & Interests** — 누가 이 결과에 관심 있는가
- **Preconditions** — 시작 전 참이어야 하는 조건
- **Success Guarantee** (Postcondition / Success End Condition) — 정상 종료 시 보장
- **Minimal Guarantee** (Failed End Condition) — 실패 시에도 보장되는 것
- **Trigger** — 무엇이 시작시키는가
- **Main Success Scenario** — 번호 매긴 정상 흐름 (1, 2, 3, ...)
- **Extensions** — 에러/대안 흐름 (3a, 3a1, 3b, ...) — Cockburn의 핵심 차별점
- **Sub-Variations** — 기술/데이터 변형 (예: 메신저 종류별)
- **Related Information (Optional)** — Priority, Performance, Frequency, Channels, Open Issues

**Gherkin (Cucumber 공식) 키워드:**

- `Feature:` — 최상위 그룹화
- `Rule:` — 비즈니스 규칙 단위(v6+)
- `Scenario:` / `Example:` — 구체 예시 한 건
- `Given` / `When` / `Then` — 초기 컨텍스트 / 이벤트 / 기대 결과
- `And` / `But` — 같은 종류의 스텝 연결
- `Background:` — Feature 전체 공유 Given
- `Scenario Outline:` + `Examples:` — 파라미터화

**UML Sequence 에러 패스 표기:**

- `alt` — 상호배타 분기 (if/else)
- `opt` — 단일 가드의 옵션 (조건 참일 때만)
- `break` — **enclosing fragment를 중단**하는 예외 시나리오 (에러 처리에 사용)
- 예외는 reply 메시지에 stereotype `<<exception>>` 표기 가능

### 2.4 DS-APM 적용 시 주의점·조정 사항

- **Level 통일**: 모든 Use Case를 Cockburn "User-goal (Sea) level"로 작성. Summary/Subfunction을 섞으면 산출물 일관성 깨짐.
- **에러 케이스 = Extension으로 시작, 복잡해지면 Gherkin으로 승격**:
  - 단순한 에러(예: "SigNoz 응답 timeout") → Cockburn Extension `3a`로 1-2줄
  - 복잡한 에러(예: "Slack webhook 4xx + 재시도 3회 실패 + Operator fallback") → 별도 Gherkin Scenario로 상세화
- **요청된 "SOP→메신저 전달 에러 케이스 1~2건"**: 다음 두 건 권장
  1. **메신저 webhook 4xx/5xx 응답 + 지수 백오프 재시도 실패**: Slack/Teams API rate limit, 인증 만료 시나리오. Extension 7a~7a3로 기록 + 시퀀스 다이어그램 `alt(success)/break(fail after retry)`.
  2. **SOP runbook 매칭 실패 (Unknown incident type)**: SOP 사전에 없는 incident가 들어왔을 때 fallback runbook 또는 Operator 직접 호출. Extension 4a로 기록 + Gherkin Scenario Outline으로 매칭 실패 종류별 분기.
- **UML use case diagram은 1장만**: 액터-유스케이스 매트릭스(스틱맨 + 타원). 너무 많이 그리면 노이즈. OMG UML v2.5.1 노테이션 준수.
- **DLQ/HMAC 미해결 follow-up**: "Open Issues" 필드에 명시.
- **Event Storming**: 워크숍에서 도메인 이벤트(주황) → 커맨드(파랑) → 액터(노랑) → 핫스팟/문제(분홍) → 정책(보라) 순으로 발굴. 최종 Use Case는 Cockburn 템플릿으로 정제.

### 2.5 HTML 렌더링 시 유리한 시각화 포인트

- **Use Case 카드 그리드**: 각 Use Case가 카드 한 장. 헤더에 Primary Actor 아이콘 + Level 배지.
- **Main Scenario ↔ Extensions 사이드바이사이드 레이아웃**: 왼쪽 컬럼은 정상 흐름(녹색), 오른쪽 컬럼은 Extension(주황/빨강). Extension 행이 정확히 해당 step 옆에 정렬 — 마크다운으로는 불가능한 공간 표현.
- **에러 시퀀스 다이어그램**: Mermaid `sequenceDiagram` + `alt/else/end`, `break` 프래그먼트. 호버하면 해당 fragment 강조.
- **Gherkin 코드 블록**: syntax highlighting (Given=파랑, When=주황, Then=녹색).
- **UML use case diagram**: SVG, 액터 클릭 시 해당 액터가 참여하는 모든 Use Case 강조.
- **Event Storming 결과물**: 가로 스크롤 타임라인 — 도메인 이벤트(주황 sticky) 좌→우 시간순 배치. 핫스팟(분홍)은 위로 튀어나오게.

### 2.6 참고 링크

- [Cockburn Use Case Template (Otago University 사본)](https://www.cs.otago.ac.nz/coursework/cosc461/uctempla.htm)
- [Cockburn Use Case Foundation (Jacobson 공저, PDF)](https://alistaircockburn.com/Use%20Case%20Foundation.pdf)
- [Cucumber Gherkin Reference (공식)](https://cucumber.io/docs/gherkin/reference/)
- [Cucumber Writing Better Gherkin](https://cucumber.io/docs/bdd/better-gherkin/)
- [UML Use Case Diagrams (uml-diagrams.org)](https://www.uml-diagrams.org/use-case-diagrams.html)
- [UML Sequence Combined Fragments](https://www.uml-diagrams.org/sequence-diagrams-combined-fragment.html)
- [EventStorming 공식](https://www.eventstorming.com/)
- [Big Picture Event Storming step-by-step](https://www.eventstormingjournal.com/big%20picture/step-by-step-guide-to-run-your-big-picture-event-storming/)

---

## 3. 기능명세서 (SRS) — 권장: **ISO/IEC/IEEE 29148 경량형 + Specification by Example**

### 3.1 권장 방법론

| 후보 | 추천 |
|---|---|
| IEEE 830-1998 | 참고만 (29148로 대체됨) |
| **ISO/IEC/IEEE 29148:2018** | **★ 채택 — 골격** |
| **Spec by Example (SBE)** | **★ 채택 — Acceptance criteria 작성법** |
| arc42 §5 Building Block | 보조 — Overview와 중복되니 SRS에는 요약만 |

**택일 추천 → ISO/IEC/IEEE 29148 SRS 템플릿의 경량화 버전을 골격으로 쓰되, 각 기능 요구사항의 acceptance criteria는 Specification by Example (Gherkin) 형식으로 작성.**

### 3.2 채택 근거

- **IEEE 830-1998은 ISO/IEC/IEEE 29148:2018로 superseded됨**. 29148이 현행 표준이며 IEEE 830의 SRS 구조를 흡수했다 (29148이 SRS, SyRS, StRS, BRS, OpsCon 5종 명세 모두 정의).
- 29148은 **good requirement의 정의**, requirement attributes/characteristics, life-cycle 적용을 모두 포함 → 한국 대기업의 감리/CMMI 대응에 단단한 기반.
- **IEEE 830 SRS 구조는 여전히 유효**(29148이 그대로 차용) → 익숙한 한국 PM/PL에 무리 없음.
- POC 단계에서 29148 전체를 따르는 건 과잉 → "29148-lite" — Section 1, 2, 3, 4만 채우고 부속 9종(traceability matrix, prototypes)은 생략.
- **Acceptance criteria는 SBE/Gherkin으로**: 요구사항 문장 "시스템은 메신저 전달 실패 시 재시도해야 한다(shall)" 은 모호하다 — Spec by Example의 "concrete example with Given/When/Then"으로 보강하면 living documentation이 된다.

### 3.3 표준이 정의하는 섹션 구조

**IEEE 830-1998 / ISO 29148 SRS 표준 outline (재사용 가능):**

**1. Introduction**
  1.1 Purpose
  1.2 Document Conventions
  1.3 Intended Audience and Reading Suggestions
  1.4 Product Scope
  1.5 References

**2. Overall Description**
  2.1 Product Perspective
  2.2 Product Features (high-level)
  2.3 User Classes and Characteristics
  2.4 Operating Environment
  2.5 Design and Implementation Constraints
  2.6 User Documentation
  2.7 Assumptions and Dependencies

**3. System Features** (intro/overview)
  3.1 [Feature Name 1]
  3.N ...

**4. External Interface Requirements**
  4.1 User Interfaces
  4.2 Hardware Interfaces
  4.3 Software Interfaces
  4.4 Communications Interfaces

**5. Other Nonfunctional Requirements**
  5.1 Performance
  5.2 Safety
  5.3 Security
  5.4 Software Quality Attributes
  5.5 Business Rules

**6. Other Requirements** / Appendices

**요구사항 표기 컨벤션** (29148/IEEE 830):

- 모든 요구사항은 `shall` 사용 (must는 의무, will은 의도, should는 권장 — 혼용 금지)
- 고유 ID: `REQ-X.X` (functional), `NF-X.X` (non-functional)
- Good requirement 속성 (29148 §5.2): Necessary, Implementation Free, Unambiguous, Consistent, Complete, Singular, Feasible, Traceable, Verifiable

**Spec by Example 7 패턴 (Gojko Adzic):**

1. Deriving scope from goals — 목표에서 범위 도출
2. Specifying collaboratively — 협업 명세
3. Illustrating requirements using examples — 예시로 요구사항 설명
4. Refining specifications — 명세 정제
5. Automating tests based on examples — 예시 기반 테스트 자동화
6. Validating frequently — 빈번한 검증
7. Evolving a documentation system — Living Documentation으로 진화

### 3.4 DS-APM 적용 시 주의점·조정 사항

- **§2.1 Product Perspective는 SigNoz 의존성을 명시**: "DS-APM is an extension layer atop SigNoz community build" — fork 프레이밍 금지, 그러나 "SigNoz 없이는 동작 불가"는 명시.
- **§3 System Features**: Overview(arc42 §5)와 중복되니 SRS에서는 1줄 요약 + 링크. 상세는 **§5 모듈별 specification** 으로 모은다 (IEEE 830 변형).
- **각 기능 요구사항에 acceptance criteria 첨부**: 예
  - `REQ-3.2.1` "시스템은 메신저 전달 실패 시 최대 3회 지수 백오프로 재시도해야 한다(shall)."
  - 그 아래 Gherkin block:
    ```gherkin
    Scenario: Slack webhook returns 429
      Given an incident with severity P1
      When the system POSTs to Slack webhook and receives 429
      Then the system shall retry after 2s, 4s, 8s
      And after 3 failed retries, the system shall enqueue to DLQ
    ```
- **§4 Communications Interfaces**: SigNoz OTel collector, ClickHouse, 메신저 webhook(REST), Operator handoff API를 모두 명시. 한국 대기업 보안팀이 "외부 통신 인터페이스 매트릭스"를 항상 요구한다.
- **§5.3 Security**: HMAC 정책 follow-up을 NF-5.3.1로 명시 — 미해결 항목도 명세에 박혀 있어야 한다.
- **§5.4 Quality Attributes**: arc42 §10의 quality tree와 cross-reference.
- **Traceability**: Use Case ↔ SRS REQ ↔ WBS work package를 ID로 연결. 29148이 강하게 요구하는 부분.
- **"shall" 일관성**: 한국어로 쓸 때 "~해야 한다"(shall) / "~할 것이다"(will) / "~할 수 있다"(may) 구분. 한국 SI 산출물이 이 부분에서 자주 흔들린다.

### 3.5 HTML 렌더링 시 유리한 시각화 포인트

- **Requirement table**: 각 행 ID/제목/우선순위/소스/상태/Verification method. 정렬·필터 가능한 인터랙티브 테이블.
- **Inline Gherkin acceptance criteria**: 각 요구사항 카드 안에 collapsible Gherkin block.
- **Traceability matrix**: Use Case × REQ × WBS 3-way 매트릭스를 sticky header 그리드로. 셀 클릭 시 양쪽 항목 강조.
- **External Interface 다이어그램**: §4를 표가 아니라 화살표 다이어그램으로. 각 인터페이스에 호버 시 프로토콜/스키마 표시.
- **NFR 카드**: Performance/Security/Reliability를 카드 셋으로, 측정 기준(p99, RPO/RTO 등) 강조.
- **Living documentation 링크**: 각 acceptance Gherkin이 실제 godog 테스트 실행 결과(pass/fail)와 연결 — Stripe docs 스타일.

### 3.6 참고 링크

- [IEEE 830-1998 Recommended Practice for SRS](https://standards.ieee.org/ieee/830/1222/)
- [ISO/IEC/IEEE 29148:2018 (ISO 페이지)](https://www.iso.org/standard/72089.html)
- [ISO/IEC/IEEE 29148-2018 (IEEE Xplore)](https://ieeexplore.ieee.org/document/8559686)
- [ISO 29148 SRS Template (ReqView)](https://www.reqview.com/doc/iso-iec-ieee-29148-templates/)
- [IEEE 830 Template Outline (오픈 교재)](https://press.rebus.community/requirementsengineering/back-matter/appendix-c-ieee-830-template/)
- [Specification by Example (Gojko Adzic)](https://gojko.net/books/specification-by-example/)
- [Specification by Example (Wikipedia, 7 patterns)](https://en.wikipedia.org/wiki/Specification_by_example)

---

## 4. WBS — 권장: **PMI Practice Standard for WBS (Hybrid 모드)**

### 4.1 권장 방법론

| 후보 | 추천 |
|---|---|
| **PMI Practice Standard for WBS (2nd ed.)** | **★ 채택 — 골격 + 100% rule** |
| **Agile Epic→Story→Task 분해** | **★ 채택 — 하위 레벨** |
| Hybrid WBS (defense materiel style) | 채택 — 상하위 결합 패턴 |

**택일 추천 → 상위 2-3 레벨(Phase/Deliverable)은 PMI WBS 100% rule을 따르는 deliverable-oriented 분해, 하위 1-2 레벨(Epic → Story/Task)은 Agile 분해. 결과적으로 hybrid WBS.**

### 4.2 채택 근거

- 한국 SI/대기업 PM은 PMI WBS에 익숙하고, 스타트업/제품팀은 Epic→Story에 익숙하다. 두 문화를 모두 경험한 작성자가 한쪽으로만 쓰면 다른 쪽 독자가 잃는다.
- PMI **100% rule** (모든 자식 = 부모의 100%, 누락/중복 없음)은 POC 산출물에서도 반드시 지켜야 할 sanity check.
- PMI **deliverable-oriented**는 "어떻게 하는가"가 아니라 "무엇을 만드는가" 중심 → SI 산출물 검수 기준과 일치.
- Agile **80-hour rule** (work package ≤ 80h) + **sprint 1-2주 단위 story**는 자연스럽게 정렬된다.
- Hybrid는 군수/방산에서 표준화된 패턴 (Wikipedia: "Elements from different WBS types can be combined to create a 'hybrid' WBS, which defense materiel project planners frequently use").

### 4.3 표준이 정의하는 섹션 구조 / 필수 요소

**PMI Practice Standard for WBS (2nd Edition) 핵심 원칙:**

1. **100% Rule** — WBS는 프로젝트 범위의 100%를 포함. 자식 요소의 합 = 부모 요소.
2. **Deliverable-Oriented** — 동사("Coding") 아닌 명사/결과물("Coded Module") 중심.
3. **Mutually Exclusive** — 요소 간 범위 겹치지 않음 (MECE).
4. **Decomposition Logic** — 일관된 분해 논리 (phase별 / 컴포넌트별 / 위치별 등 한 가지 기준).
5. **Level of Detail** — 보통 2-4 레벨. 80-hour rule, reporting-period rule, common sense rule.
6. **WBS Dictionary** — 각 요소의 deliverables, activities, milestones, resources, cost, quality를 기술한 별도 문서.
7. **Work Package** — 최하위 terminal element. 비용/기간 추정 가능 단위.

**Agile 하위 분해 (Atlassian/일반 통용):**

- **Initiative** → **Epic** → **Story** → **Task** / **Subtask**
- **Epic**: 여러 sprint에 걸친 큰 작업, 여러 팀에 분산 가능
- **Story**: 1 sprint 내 완료 가능한 단위, INVEST 원칙 (Independent, Negotiable, Valuable, Estimable, Small, Testable)
- **Task**: 기술적 실행 단위 (보통 시간 단위)

**Hybrid 매핑 (DS-APM 권장):**

```
Level 1: DS-APM Project (root)
Level 2: Phase / Major Deliverable (PMI deliverable)
   예: "POC", "Beta", "GA"
   또는 컴포넌트별: "Incident Ingest", "SOP Engine", "Messenger Adapter"
Level 3: Sub-deliverable / Epic (전환점)
   예: "Slack Adapter v1"
Level 4: Story (Agile, 1 sprint)
   예: "Slack webhook 4xx 재시도 로직 구현"
Level 5: Task (옵션, 시간 단위)
```

### 4.4 DS-APM 적용 시 주의점·조정 사항

- **Decomposition logic 통일**: Level 2를 phase로 할지 component로 할지 한 가지로 통일. POC 단계는 **component-oriented** 권장 (Incident Ingest / SOP Engine / Messenger Adapter / Operator Handoff / Observability Plumbing).
- **100% rule 검증**: 작성 후 "이 5개 컴포넌트로 DS-APM의 모든 일이 다 커버되는가?" 자가 검증. SigNoz 위에 얹는 레이어이므로 "SigNoz 자체 운영"은 명시적으로 **OUT OF SCOPE**로 빼야 한다.
- **WBS Dictionary는 별도 페이지**: 각 work package에 대해 "Deliverable / Acceptance criteria / Owner / Estimated effort / Dependencies / Verification". Acceptance criteria는 §3 SRS와 traceability ID로 연결.
- **80-hour rule**: 80h 넘는 task는 더 쪼개라. 단, POC에서 너무 잘게 쪼개면 유지비용 폭증 → 상위만 단단히, 하위는 Just-In-Time 분해.
- **사용자 메모리 정책 반영**:
  - "y2i 영구 비활성화" → WBS에서 y2i 관련 work package 제거, 명시적 "Excluded scope" 섹션에 기록.
  - "Orchestrator → SigNoz 마이그레이션 follow-up (DLQ 활성화 + HMAC 정책)" → 별도 Epic으로 등재.
- **WBS Code 부여**: `1.2.3` 형태. 한국 SI 감리/검수에서 ID 없는 WBS는 통과 불가.
- **Milestone 표시**: Level 2/3 사이에 Milestone diamond (PMI 표기) — POC 완료, Beta GA 등.
- **Status는 WBS 자체에 박지 말기**: 진행률은 별도 burndown/dashboard. WBS는 "scope of work" 정적 문서, 진행은 동적.

### 4.5 HTML 렌더링 시 유리한 시각화 포인트

- **Indented tree + 트리뷰**: collapsible 1.x → 1.1.x → 1.1.1.x. 각 노드에 ID/제목/effort/owner.
- **Sunburst chart / Treemap**: 100% rule 시각 검증에 최적 — 부모 100% 안에서 자식이 어떻게 분할되는지 한눈에.
- **Gantt chart (옵션)**: WBS work package를 시간축에 매핑. dependency 화살표.
- **WBS Dictionary side panel**: 트리에서 노드 클릭 시 우측 패널에 dictionary entry. Acceptance criteria, SRS 링크, Use Case 링크.
- **Traceability heatmap**: WBS × SRS × Use Case 3-way 매트릭스 — 커버리지 빈 셀이 빨갛게 표시.
- **Excluded scope 박스**: 명시적 OUT OF SCOPE 항목을 회색/취소선 박스로 분리 (100% rule 보조).
- **Hybrid 표시**: Level 2-3은 PMI 박스 스타일(직사각형 + ID), Level 4-5는 Agile 카드 스타일(둥근 모서리 + 색 라벨). 시각적으로 hybrid임이 드러남.

### 4.6 참고 링크

- [PMI Practice Standard for WBS (PMI Library)](https://www.pmi.org/learning/library/practice-standard-work-breakdown-structures-8063)
- [WBS — Wikipedia (100% rule, work packages)](https://en.wikipedia.org/wiki/Work_breakdown_structure)
- [The ABC basics of WBS (Paul Burek, PMI)](https://www.pmi.org/learning/library/work-breakdown-structure-basics-5919)
- [Atlassian — Epics, Stories, Initiatives](https://www.atlassian.com/agile/project-management/epics-stories-themes)
- [Hybrid WBS in Agile (Miro)](https://miro.com/project-management/work-breakdown-structure-agile/)

---

## 5. 산출물 4종 간 Traceability — 통합 권장사항

4종 산출물은 따로 노는 문서가 아니라 **하나의 traceability chain**으로 연결되어야 한다.

```
Overview (arc42 §3 Context)
   └── 정의된 시스템 경계
         └── Use Case (UC-001 ~ UC-N) — Cockburn template
               └── Main scenario step → SRS functional REQ-X.X
                    └── Acceptance criteria (Gherkin)
                         └── godog/Cucumber 실행 → Living Documentation
               └── Extension (error path) → SRS NF-X.X (resilience)
                                          → WBS work package
         └── WBS root
               └── Component (Level 2, deliverable)
                    └── Epic (Level 3)
                         └── Story (Level 4) — implements REQ-X.X
```

**HTML 출력 시 cross-link 권장사항:**

- 각 ID(`UC-001`, `REQ-3.2.1`, `WBS-1.2.3`)는 클릭 가능한 anchor.
- 호버 시 양방향 참조 표시 (이 REQ를 구현하는 WBS는? 이 WBS가 충족하는 Use Case는?).
- 4종 문서가 단일 사이트(또는 단일 HTML)에 있으면 단순 `#anchor`, 다중 파일이면 명시적 cross-document link.

---

## 출처 (Sources)

### Overview
- [arc42 Template Overview (공식)](https://arc42.org/overview)
- [arc42 Documentation](https://docs.arc42.org/home/)
- [arc42 §5 Building Block View](https://docs.arc42.org/section-5/)
- [arc42 Building Block Examples](https://docs.arc42.org/examples/)
- [C4 model (Simon Brown)](https://c4model.com/)
- [Diátaxis Framework](https://diataxis.fr/)
- [Make a README](https://www.makeareadme.com/)

### Use Case
- [Cockburn Use Case Template (Otago)](https://www.cs.otago.ac.nz/coursework/cosc461/uctempla.htm)
- [Use Case Foundation (Cockburn & Jacobson)](https://alistaircockburn.com/Use%20Case%20Foundation.pdf)
- [Cucumber Gherkin Reference](https://cucumber.io/docs/gherkin/reference/)
- [Cucumber Writing Better Gherkin](https://cucumber.io/docs/bdd/better-gherkin/)
- [UML Use Case Diagrams](https://www.uml-diagrams.org/use-case-diagrams.html)
- [UML Sequence Combined Fragments](https://www.uml-diagrams.org/sequence-diagrams-combined-fragment.html)
- [EventStorming 공식](https://www.eventstorming.com/)
- [Big Picture EventStorming step-by-step](https://www.eventstormingjournal.com/big%20picture/step-by-step-guide-to-run-your-big-picture-event-storming/)
- [Event Storming — Wikipedia](https://en.wikipedia.org/wiki/Event_storming)

### SRS
- [IEEE 830-1998 (IEEE SA)](https://standards.ieee.org/ieee/830/1222/)
- [ISO/IEC/IEEE 29148:2018 (ISO)](https://www.iso.org/standard/72089.html)
- [ISO/IEC/IEEE 29148-2018 (IEEE Xplore)](https://ieeexplore.ieee.org/document/8559686)
- [ISO 29148 Templates (ReqView)](https://www.reqview.com/doc/iso-iec-ieee-29148-templates/)
- [IEEE 830 Template Outline](https://press.rebus.community/requirementsengineering/back-matter/appendix-c-ieee-830-template/)
- [Specification by Example (Gojko Adzic)](https://gojko.net/books/specification-by-example/)
- [Specification by Example — Wikipedia (7 patterns)](https://en.wikipedia.org/wiki/Specification_by_example)

### WBS
- [PMI Practice Standard for WBS](https://www.pmi.org/learning/library/practice-standard-work-breakdown-structures-8063)
- [WBS — Wikipedia](https://en.wikipedia.org/wiki/Work_breakdown_structure)
- [WBS Basics (Paul Burek, PMI)](https://www.pmi.org/learning/library/work-breakdown-structure-basics-5919)
- [Atlassian — Epics/Stories/Initiatives](https://www.atlassian.com/agile/project-management/epics-stories-themes)
- [Hybrid WBS (Miro)](https://miro.com/project-management/work-breakdown-structure-agile/)

### 공통/HTML
- [HTML Effectiveness (Thariq Shihipar)](https://thariqs.github.io/html-effectiveness/)
- [Runbook vs SOP (Upstat)](https://upstat.io/blog/runbook-vs-sop)
- [SigNoz Docs](https://signoz.io/docs/introduction/)

---

**리서치 검증 노트**: arc42 v9.0 (2025년 7월) §10 재구조화, ISO/IEC/IEEE 29148:2018 (IEEE 830 superseded), Cockburn extension 번호 규칙(3a, 3a1), Gherkin v6 `Rule` 키워드, PMI WBS 2nd Edition 100% rule, UML 2.5.1 sequence `alt/opt/break` fragment — 모두 공식 문서로 확인됨. Event Storming의 전체 색상 컨벤션(특히 보라/녹색 stickies)은 단일 공식 출처로 완전 확정 못함 → Wikipedia + EventStormingJournal 조합으로 확정한 부분만 기재.
