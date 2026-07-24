// 히스토그램 버킷 기하 유틸. 막대는 [edge, edge+bucketSize) 구간을 채우므로
// hover 매핑은 최근접이 아니라 bracket(floor) 탐색이어야 한다(스펙 R2).
// prepareHistogramPanelData가 맨 앞에 넣는 유령 널빈(count=null)은 스킵한다(R3).
//
// 주의(M2): 엣지 배열은 균일 격자가 아니다. buildHistogramBuckets가 값이 있는
// 버킷만 만들어서 빈 버킷은 배열에서 빠진다. 따라서 후보 버킷을 찾은 뒤
// edges[i] + bucketSize 상한을 반드시 확인해야 빈 구간이 직전 버킷으로
// 오귀속되지 않는다.
// 주의(M3): 실제 입력은 axisPointer.snap이 스냅한 "버킷 중심"값이다.
// 중심값도 bracket으로 같은 버킷이 나오므로 이 함수는 두 형태 모두 처리한다.

/**
 * 버킷 엣지 배열에서 균일 버킷 폭을 얻는다. 엣지가 2개 미만이면 0.
 * 엣지 배열은 빈 버킷이 빠진 비균일(구멍 있는) 배열일 수 있으므로
 * 단순히 edges[1]-edges[0]을 쓰지 않고, 인접 엣지 간 최소 양수 간격을
 * 취해서 실제 버킷 폭을 구한다(sparse-safe).
 */
export function deriveBucketSize(edges: number[]): number {
	if (edges.length < 2) {
		return 0;
	}
	let minGap = Infinity;
	for (let i = 0; i < edges.length - 1; i += 1) {
		const gap = edges[i + 1] - edges[i];
		if (gap > 0 && gap < minGap) {
			minGap = gap;
		}
	}
	return Number.isFinite(minGap) ? minGap : 0;
}

/**
 * axisPointer x값 → 버킷 인덱스(bracket 탐색).
 * 각 버킷은 [edges[i], edges[i]+bucketSize) 구간만 유효하고, 그 밖(빈 버킷 구간·
 * 마지막 버킷 오른쪽)은 null이다.
 */
export function resolveBucketIndex(
	x: number,
	edges: number[],
	options?: { skipLeadingNullBin?: boolean },
): number | null {
	if (edges.length === 0) {
		return null;
	}
	const bucketSize = deriveBucketSize(edges);
	if (x < edges[0]) {
		return null;
	}

	// x 이하인 마지막 엣지 = 후보 버킷 (엣지는 오름차순)
	let index = edges.length - 1;
	for (let i = 0; i < edges.length - 1; i += 1) {
		if (x < edges[i + 1]) {
			index = i;
			break;
		}
	}

	// 상한 확인 — 빈 버킷이 빠진 비균일 배열이라 후보와 x가 한 버킷 이상 떨어질 수
	// 있다(M2). bucketSize가 0(엣지 1개)이면 그 엣지에 정확히 일치할 때만 유효.
	const isInsideBucket =
		bucketSize > 0 ? x < edges[index] + bucketSize : x === edges[index];
	if (!isInsideBucket) {
		return null;
	}

	// 선행 유령 널빈은 카운트가 null이라 툴팁이 비므로 스킵 규약을 적용한다
	if (options?.skipLeadingNullBin && index === 0) {
		return null;
	}
	return index;
}
