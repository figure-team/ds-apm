/* eslint-disable */
// One-off generator for a sample SOP bulk-upload xlsx.
// Columns must match parseSopExcel.ts REQUIRED_COLUMNS + optional ones.
const XLSX = require('xlsx');
const path = require('path');

const headers = [
	'sop_id',
	'title',
	'version',
	'owner_team',
	'approval_status',
	'source_id',
	'project_ids',
	'environments',
	'display_url',
	'tags',
	'service_account_profile',
	'body_markdown',
	'customer_update_template',
	'vendor_request_template',
];

// project_ids / environments are the SOP binding scope (alert labels).
// Adjust these to match your real alert labels if needed.
const PROJECT = 'otel-demo';
const ENV = 'prod';

function row(o) {
	return [
		o.sop_id,
		o.title,
		o.version,
		o.owner_team,
		o.approval_status,
		'src-managed-markdown-default',
		PROJECT,
		ENV,
		o.display_url || '',
		o.tags || '',
		'managed-markdown-local',
		o.body_markdown,
		o.customer || '',
		o.vendor || '',
	];
}

const data = [
	{
		sop_id: 'SOP-AD-001',
		title: 'Ad 서비스 응답 지연/오류 대응',
		version: '2026-06-01.1',
		owner_team: 'ads',
		approval_status: 'approved',
		tags: 'ad,latency',
		body_markdown:
			'# Ad 서비스 응답 지연/오류\n\n1. ad 서비스의 p95 레이턴시 대시보드 확인\n2. 광고 추천 백엔드/캐시 연결 상태 점검\n3. 최근 배포 이력 확인 후 필요 시 롤백\n4. 트래픽 급증 시 HPA/레플리카 스케일 조정',
		customer: '[안내] 광고 영역 일시 지연이 발생했습니다. 조치 중이며 곧 정상화됩니다.',
	},
	{
		sop_id: 'SOP-CART-001',
		title: 'Cart 서비스 장바구니 오류 대응',
		version: '2026-06-01.1',
		owner_team: 'cart',
		approval_status: 'approved',
		tags: 'cart,redis',
		body_markdown:
			'# Cart 서비스 장바구니 오류\n\n1. cart 서비스 에러율/5xx 추이 확인\n2. Redis(valkey) 연결 및 메모리 상태 점검\n3. 장바구니 read/write 타임아웃 로그 확인\n4. Redis 장애 시 페일오버 후 캐시 워밍업',
	},
	{
		sop_id: 'SOP-CHECKOUT-001',
		title: 'Checkout 결제 체크아웃 실패 대응',
		version: '2026-06-01.1',
		owner_team: 'checkout',
		approval_status: 'approved',
		tags: 'checkout,critical',
		body_markdown:
			'# Checkout 결제 체크아웃 실패\n\n1. checkout 서비스 주문 실패율 확인\n2. 의존 서비스(cart, payment, currency, email) 상태 점검\n3. 결제 프로바이더 연동 타임아웃 로그 확인\n4. 미완료 주문 보정 배치 필요 여부 판단',
		customer: '[안내] 일부 결제 처리가 지연되고 있습니다. 중복 결제는 발생하지 않으며 자동 보정됩니다.',
		vendor: '안녕하세요. {서비스} 결제 연동에서 {증상} 확인됩니다. 트랜잭션 ID {ID} 상태 확인 부탁드립니다.',
	},
	{
		sop_id: 'SOP-CURRENCY-001',
		title: 'Currency 환율 변환 서비스 오류 대응',
		version: '2026-06-01.1',
		owner_team: 'platform',
		approval_status: 'draft',
		tags: 'currency',
		body_markdown:
			'# Currency 환율 변환 오류\n\n1. currency 서비스 gRPC 에러율 확인\n2. 환율 데이터 소스/캐시 만료 여부 점검\n3. checkout 등 다운스트림 영향 범위 확인\n4. 변환 실패 시 마지막 정상 환율로 폴백',
	},
	{
		sop_id: 'SOP-EMAIL-001',
		title: 'Email 발송 실패 대응',
		version: '2026-06-01.1',
		owner_team: 'platform',
		approval_status: 'approved',
		tags: 'email,notification',
		body_markdown:
			'# Email 발송 실패\n\n1. email 서비스 발송 실패율/큐 적체 확인\n2. SMTP/메일 게이트웨이 연결 상태 점검\n3. 주문 확인 메일 누락분 재발송 대상 식별\n4. 게이트웨이 장애 시 보조 채널 전환',
	},
	{
		sop_id: 'SOP-FRONTENDPROXY-001',
		title: 'Frontend-proxy(Envoy) 5xx 대응',
		version: '2026-06-01.1',
		owner_team: 'platform',
		approval_status: 'approved',
		tags: 'frontend-proxy,envoy,5xx',
		body_markdown:
			'# Frontend-proxy 5xx 급증\n\n1. frontend-proxy의 upstream 5xx/연결 실패율 확인\n2. 라우팅 대상(web, frontend) 헬스체크 상태 점검\n3. Envoy circuit breaker/리트라이 설정 확인\n4. 비정상 업스트림 격리 후 트래픽 재분배',
		customer: '[안내] 서비스 접속이 일시적으로 불안정합니다. 빠르게 복구 중입니다.',
	},
	{
		sop_id: 'SOP-WEB-001',
		title: 'Web 프론트엔드 로딩 오류 대응',
		version: '2026-06-01.1',
		owner_team: 'frontend',
		approval_status: 'draft',
		tags: 'web,frontend',
		body_markdown:
			'# Web 프론트엔드 로딩 오류\n\n1. web 서비스 페이지 로드 실패율/JS 에러 확인\n2. API(frontend-proxy 경유) 응답 상태 점검\n3. 정적 자산 CDN/캐시 상태 확인\n4. 최근 프론트 배포 롤백 여부 판단',
	},
	{
		sop_id: 'SOP-PROVIDER-001',
		title: 'Provider(외부 연동) 장애 대응',
		version: '2026-06-01.1',
		owner_team: 'payments',
		approval_status: 'approved',
		tags: 'provider,vendor',
		body_markdown:
			'# Provider 외부 연동 장애\n\n1. provider 연동 에러율/타임아웃 추이 확인\n2. 외부 API 상태 페이지/공지 확인\n3. 영향 받는 거래 범위 식별 및 재시도 정책 점검\n4. 장기 장애 시 공급사에 정식 문의 발송',
		vendor: '안녕하세요. {서비스} 연동에서 {증상}이 확인됩니다. 장애 범위와 예상 복구 시점 공유 부탁드립니다.',
	},
	{
		sop_id: 'SOP-LOADGEN-001',
		title: 'Load-generator 이상 트래픽 대응',
		version: '2026-06-01.1',
		owner_team: 'sre',
		approval_status: 'disabled',
		tags: 'load-generator,test',
		body_markdown:
			'# Load-generator 이상 트래픽\n\n1. load-generator의 RPS/시나리오 설정 확인\n2. 의도치 않은 부하가 prod에 유입되는지 점검\n3. 필요 시 부하 생성기 일시 중지\n4. 테스트 트래픽과 실사용 트래픽 분리 검증',
	},
	{
		sop_id: 'SOP-PLATFORM-001',
		title: '전반적 고지연/인프라 포화 대응',
		version: '2026-06-01.1',
		owner_team: 'sre',
		approval_status: 'deprecated',
		tags: 'platform,infra,latency',
		body_markdown:
			'# 전반적 고지연 / 인프라 포화\n\n1. 클러스터 CPU/메모리/네트워크 포화 여부 확인\n2. 공통 의존성(DB, 캐시, 메시지큐) 상태 점검\n3. 영향 서비스 우선순위에 따라 스케일 조정\n4. 근본 원인 RCA 티켓 생성 및 사후 분석',
	},
];

const aoa = [headers, ...data.map(row)];
const ws = XLSX.utils.aoa_to_sheet(aoa);
ws['!cols'] = [
	{ wch: 22 },
	{ wch: 34 },
	{ wch: 14 },
	{ wch: 14 },
	{ wch: 16 },
	{ wch: 30 },
	{ wch: 14 },
	{ wch: 12 },
	{ wch: 30 },
	{ wch: 24 },
	{ wch: 24 },
	{ wch: 70 },
	{ wch: 50 },
	{ wch: 50 },
];
const wb = XLSX.utils.book_new();
XLSX.utils.book_append_sheet(wb, ws, 'SOP Sample');
const out = path.resolve(process.argv[2] || 'sop-sample-10.xlsx');
XLSX.writeFile(wb, out);
console.log('wrote', out, '-', data.length, 'rows');
