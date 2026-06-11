# i18n 한국어 번역 용어집 (Glossary)

모든 한국어 번역은 아래 표준 용어를 따른다. (출처: 기존 ko/routes.json, ko/settings.json)

| English | 한국어 | English | 한국어 |
|---|---|---|---|
| Home | 홈 | Save | 저장 |
| Alerts | 알림 | Cancel | 취소 |
| Dashboards | 대시보드 | Edit | 편집 |
| Services | 서비스 | Delete | 삭제 |
| Logs | 로그 | Create | 생성 |
| Traces | 트레이스 | Add | 추가 |
| Metrics | 메트릭 | Update | 업데이트 |
| Infrastructure | 인프라 | Search | 검색 |
| Exceptions | 예외 | Filter | 필터 |
| Messaging Queues | 메시지 큐 | Apply | 적용 |
| Service Map | 서비스 맵 | Close | 닫기 |
| Settings | 설정 | Confirm | 확인 |
| General | 일반 | Refresh | 새로고침 |
| Workspace | 워크스페이스 | Export | 내보내기 |
| Members | 멤버 | Import | 가져오기 |
| Billing | 결제 | Copy | 복사 |
| Integrations | 통합 | Name | 이름 |
| Pipeline | 파이프라인 | Description | 설명 |
| API Keys | API 키 | Status | 상태 |
| Account | 계정 | Actions | 작업 |
| Password | 비밀번호 | Loading | 로딩 중 |
| Organization | 조직 | Next / Back | 다음 / 이전 |
| Dashboard | 대시보드 | Enabled / Disabled | 활성화 / 비활성화 |

## 규칙

- 키는 절대 바꾸지 않는다. 값(영문)만 한국어로 번역한다.
- 보간 변수 `{{var}}`, HTML 태그 `<tag>`, 고유명사는 그대로 보존한다.
- 고유명사·기술용어는 영문 유지: SigNoz, SSO, MCP, SOP, API, URL, JSON, Slack, Webhook, OpenTelemetry, Kafka 등.
  - 예외: 페이지/탭 제목으로 쓰일 때는 공식 한글 표기를 따른다 — Kubernetes → 쿠버네티스, Hosts → 호스트.
- 이메일/도메인 형식 예시(placeholder)는 영문 유지 가능.

## 공통 UI 문구 (common.json) — 페이지 전환 시 재사용

전역 공유 컴포넌트의 문구는 `public/locales/{en,ko}/common.json`에 이미 키가 있다.
**새 페이지를 한글 전환할 때 아래 컴포넌트를 쓰고 있다면 추가 작업이 필요 없다**
(표시 시점에 자동 번역됨). 같은 문구를 페이지별 네임스페이스에 중복 추가하지 말 것.

| 컴포넌트 | 문구 | 키 |
|---|---|---|
| 시간 범위 선택기 (`CustomTimePicker`, TopNav `DateTimeSelectionV2`) | "Last 30 minutes" 등 옵션 라벨 전체 | `common:time_range.*` |
| 〃 | "Select / Enter Time Range", "RELATIVE TIMES", "RECENTLY USED", "Change Timezone", "Search timezones...", Zoom out 툴팁 | `common:select_enter_time_range` 등 |
| 쿼리 빌더 검색 (`QueryBuilderSearch`, `QueryBuilderSearchV2`, `ClientSideQBSearch`) | "Search Filter : select options from..." 기본 placeholder | `common:qb_search_placeholder` |

구현 패턴:

- 시간 범위 라벨은 `Options` 상수(영문)를 그대로 두고 표시 시점에만
  `translateTimeRangeLabel(t, label)` (`container/TopNav/DateTimeSelectionV2/constants.ts`)로 번역한다.
  value('30m' 등)와 영문 라벨은 URL/redux/비교 로직에 쓰이므로 절대 번역하지 않는다.
- QB 검색 placeholder는 `PLACEHOLDER` 상수를 기본 prop 센티널로 유지하고,
  컴포넌트 내부에서 `placeholder === PLACEHOLDER ? t('qb_search_placeholder') : placeholder`로 해석한다.
  페이지가 커스텀 placeholder를 넘기면 그 문구는 해당 페이지 네임스페이스에서 번역할 것.
- 모듈 스코프 `TabRoutes`의 `name`(JSX)은 훅을 못 쓰므로 라벨을 작은 컴포넌트로 분리해
  내부에서 `useTranslation`을 호출한다 (예: `pages/InfrastructureMonitoring/constants.tsx`의 `TabName`).
- 테스트는 i18n 리소스가 로드되지 않아 `t()`가 키를 그대로 반환한다 —
  문구 단언은 키 기준으로 작성한다 (예: `getByText('qb_search_placeholder')`).
