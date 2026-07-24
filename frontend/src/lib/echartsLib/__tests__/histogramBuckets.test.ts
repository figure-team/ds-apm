import {
	deriveBucketSize,
	resolveBucketIndex,
} from '../utils/histogramBuckets';

// 유령 널빈(-10) + 실제 버킷 0,10,...,40. bucketSize=10
const edges = [-10, 0, 10, 20, 30, 40];

describe('deriveBucketSize', () => {
	it('균일 엣지 간격을 반환한다', () => {
		expect(deriveBucketSize(edges)).toBe(10);
	});

	it('엣지가 2개 미만이면 0', () => {
		expect(deriveBucketSize([])).toBe(0);
		expect(deriveBucketSize([5])).toBe(0);
	});
});

describe('resolveBucketIndex (bracket 탐색)', () => {
	it('버킷 좌측 경계는 그 버킷', () => {
		expect(resolveBucketIndex(10, edges)).toBe(2);
	});

	it('버킷 우측 절반도 같은 버킷 (최근접 엣지 아님 — R2 핵심)', () => {
		// x=18은 [10,20) 구간 → index 2. 최근접 엣지라면 20(index 3)으로 오귀속된다
		expect(resolveBucketIndex(18, edges)).toBe(2);
		expect(resolveBucketIndex(19.99, edges)).toBe(2);
	});

	it('다음 버킷 경계는 다음 버킷', () => {
		expect(resolveBucketIndex(20, edges)).toBe(3);
	});

	// ⚠️ 프로덕션에서 실제로 들어오는 입력은 이 케이스다(M3).
	// axisPointer.snap:true라 페이로드 value가 버킷 중심으로 스냅되어 도착한다.
	it('스냅된 버킷 중심값을 그 버킷으로 해석한다 — 실제 입력 형태', () => {
		expect(resolveBucketIndex(5, edges)).toBe(1); // [0,10) 중심
		expect(resolveBucketIndex(15, edges)).toBe(2); // [10,20) 중심
		expect(resolveBucketIndex(45, edges)).toBe(5); // 마지막 버킷 중심
		// 유령 널빈 중심(-5)은 스킵 규약 대상
		expect(resolveBucketIndex(-5, edges, { skipLeadingNullBin: true })).toBeNull();
	});

	it('마지막 버킷 내부는 유효, 우측 한계를 넘으면 null', () => {
		expect(resolveBucketIndex(49.9, edges)).toBe(5);
		expect(resolveBucketIndex(50, edges)).toBeNull();
	});

	it('마지막 버킷을 넘어서면 null', () => {
		expect(resolveBucketIndex(1000, edges)).toBeNull();
	});

	it('선행 유령 널빈(index 0) hover는 스킵한다 — R3', () => {
		// x=-5는 [-10,0) = index 0(유령) → 스킵 규약상 null
		expect(resolveBucketIndex(-5, edges, { skipLeadingNullBin: true })).toBeNull();
		// 스킵 미지정이면 index 0을 그대로 반환
		expect(resolveBucketIndex(-5, edges)).toBe(0);
	});

	it('첫 엣지보다 왼쪽이면 null', () => {
		expect(resolveBucketIndex(-50, edges)).toBeNull();
	});

	it('엣지가 비면 null', () => {
		expect(resolveBucketIndex(5, [])).toBeNull();
	});
});

// M2: 값이 없는 버킷은 엣지 배열에서 통째로 빠진다(buildHistogramBuckets).
// 상한 확인 없이 bracket만 하면 빈 구간이 직전 버킷으로 오귀속된다.
describe('resolveBucketIndex (비균일 엣지 — 빈 버킷 누락)', () => {
	// 20번 버킷에 값이 없어 빠진 배열. bucketSize는 여전히 10
	const gapped = [-10, 0, 10, 30, 40];

	it('빈 구간(20~30) hover는 직전 버킷이 아니라 null', () => {
		expect(resolveBucketIndex(25, gapped)).toBeNull();
		expect(resolveBucketIndex(20, gapped)).toBeNull();
	});

	it('구멍 양옆의 실제 버킷은 정상 해석', () => {
		expect(resolveBucketIndex(15, gapped)).toBe(2); // [10,20)
		expect(resolveBucketIndex(35, gapped)).toBe(3); // [30,40)
		expect(resolveBucketIndex(45, gapped)).toBe(4); // [40,50)
	});
});
