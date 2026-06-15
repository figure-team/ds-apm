# CF-2 고객/벤더 커뮤니케이션 템플릿 그라운딩 (설계 노트)

- 상태: **Implemented — 전용 필드** (2026-06-15). FE 입력칸 포함 백엔드 완결.
- 작성일: 2026-06-15
- 관련: CF-2 (AI 1차 분석 초안), `customerUpdateDraft` / `vendorRequestDraft`
- 선행 완료(A): `llmaigenerator/prompt.go` — customerUpdateDraft에 공지문 양식(제목+■라벨) + 가드(원인 단정·배상/법적책임·확정 ETA 금지)

## 구현 결과 (전용 필드 방식 채택)

스키마 택1 중 **(1) 전용 필드**로 구현. SOP는 payload JSON 블롭 저장이라 마이그레이션·contractVersion bump 불필요(additive·하위호환).

| 레이어 | 변경 |
|---|---|
| 타입 | `SOPDocument.CustomerUpdateTemplate` / `VendorRequestTemplate` (`json:omitempty`) |
| 프롬프트 | `renderUser`가 템플릿 렌더 + systemMessage: "템플릿 있으면 {슬롯}만 채우고 문구·구조 유지, 없으면 공지문 양식 자유작성" |
| FE | SOP 편집(`SOPDocuments.tsx`)에 "고객 공지 템플릿"·"공급사 요청 템플릿" TextArea + i18n(ko/en) |
| 테스트 | 프롬프트 골든 + 템플릿 렌더 테스트 / 백엔드 green |

**실증(codex)**: 조직 고정 문구(제목·`문의처 1588-0000`)는 verbatim 유지, `{슬롯}`만 인시던트 정보로 채움 → 슬랙 전송 확인. 핵심 가치(브랜드/연락처/법적경계 통제 + LLM은 변수만) 입증됨.

남은(선택): 템플릿 슬롯 규약 문서화, SOP approval과 템플릿 변경 거버넌스 연계.

---
(이하 원 설계 노트)

## 배경 / 문제

`customerUpdateDraft`·`vendorRequestDraft`는 현재 LLM이 **자유 생성**한다. A 작업으로 구조·가드는 프롬프트에 박았으나, 여전히:

- 톤·문구가 인시던트마다 모델 재량 → 조직이 **승인한 브랜드 문구**가 아님
- 법적/컴플라이언스 문구(면책·SLA 표현 등)를 프롬프트 가드에만 의존 → 통제가 약함
- 조직/서비스별로 다른 공지 양식을 반영할 수 없음

고객 대면 문구는 초안(HITL, 자동 발송 아님)이라도 **일관성·통제**가 필요하다.

## 제안 (B): SOP에 comms 템플릿 섹션을 얹고 그라운딩

순수 LLM도 순수 고정 템플릿도 아닌 **하이브리드**: 템플릿이 뼈대(구조 + 승인된 문구)를 잡고, LLM은 인시던트 변수(슬롯)만 채운다. 이미 있는 **SOP 그라운딩 메커니즘을 그대로 재사용**한다.

### 핵심 아이디어
- SOP 문서(`ruletypes.SOPDocument`)에 **고객 공지 템플릿 / 벤더 요청 템플릿** 필드(또는 bodyMarkdown 내 규약 섹션)를 추가
- 프롬프트가 SOP 단계를 그라운딩하듯, 이 템플릿을 그라운딩해 슬롯만 채우게 지시
- 템플릿이 비어 있으면 현재(A) 동작(구조+가드 기반 자유 생성)으로 폴백

### 스키마 방향 (택1, 미결)
1. `SOPDocument`에 구조화 필드 추가: `CustomerUpdateTemplate string`, `VendorRequestTemplate string` (+ contractVersion bump)
2. bodyMarkdown 내 규약 헤딩(예: `## 고객 공지 템플릿`)을 파서가 추출 — 스키마 무변경, 규약 의존
3. 별도 `comms_template` 리소스(org/service 스코프) + SOP가 참조

### 프롬프트 변경
- 템플릿이 주어지면: "아래 고객 공지 템플릿의 `{슬롯}`만 인시던트 정보로 채우고, 그 외 문구·구조·순서는 변경 금지"
- 슬롯 규약 정의(`{영향범위}`, `{다음안내시각}` 등)

## 기존 자산과의 정렬

- **`/api/v2/rules/notification_template/preview`** (`PreviewNotificationTemplate`) — 채널 렌더링 템플릿. comms 템플릿(=초안 내용)과 **레이어가 다름**: comms 템플릿은 "무슨 말을 쓸지", notification 템플릿은 "채널에 어떻게 렌더할지". 둘을 연결/분리 정책 필요.
- **SOP = 진실의 원천** — comms 템플릿을 SOP에 얹으면 조직 승인 경로(SOP approval)가 그대로 적용됨.
- 디스패치 경로의 **evidence 미전달(v0.1 갭)** 과는 독립 과제. (그 갭은 별도: EvidenceCollector 배선)

## 오픈 퀘스천
- 템플릿 스코프: SOP별 vs org별 vs service별?
- 다국어(고객 공지 언어) 정책
- 슬롯 미충족 시(근거 부족) 동작 — 슬롯 비우기 vs "확인 중" 표기
- 승인 워크플로우: comms 템플릿도 SOP approval에 종속시킬지

## 범위 / 비용
- 스키마(택1) + 프롬프트 그라운딩 + 검증 + 골든 테스트. 중간 규모.
- A(프롬프트 구조+가드, 완료)로 단기 품질은 확보됐으므로 **우선순위는 중**.
