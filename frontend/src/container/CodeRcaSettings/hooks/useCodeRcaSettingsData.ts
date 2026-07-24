import { type Dispatch, type SetStateAction, useEffect, useState } from 'react';
import getConfig from 'api/codeRca/getConfig';
import listRepos from 'api/codeRca/listRepos';
import listServiceMaps from 'api/codeRca/listServiceMaps';
import {
	CodebaseRepo,
	CodebaseServiceMap,
	CodeRcaConfig,
} from 'api/codeRca/types';

type CodeRcaSettingsData = {
	config: CodeRcaConfig | null;
	setConfig: Dispatch<SetStateAction<CodeRcaConfig | null>>;
	repos: CodebaseRepo[];
	setRepos: Dispatch<SetStateAction<CodebaseRepo[]>>;
	serviceMaps: CodebaseServiceMap[];
	setServiceMaps: Dispatch<SetStateAction<CodebaseServiceMap[]>>;
};

/**
 * Code RCA 설정 탭의 세 데이터(설정·저장소·서비스 매핑)를 한 곳에서 싣는다.
 *
 * 세 로드를 도메인별 훅으로 쪼개지 않는 것은 의도적이다 — 지금은 하나의
 * Promise.all + 공유 cancelled 플래그라서 셋 중 하나만 실패해도 나머지도
 * setState되지 않는다(all-or-nothing). 쪼개면 부분 성공으로 동작이 바뀐다.
 * 도메인 분리는 렌더·핸들러 계층에서만 한다.
 */
export function useCodeRcaSettingsData(): CodeRcaSettingsData {
	const [config, setConfig] = useState<CodeRcaConfig | null>(null);
	const [repos, setRepos] = useState<CodebaseRepo[]>([]);
	const [serviceMaps, setServiceMaps] = useState<CodebaseServiceMap[]>([]);

	useEffect(() => {
		let cancelled = false;

		const load = async (): Promise<void> => {
			try {
				const [cfgRes, reposRes, mapsRes] = await Promise.all([
					getConfig(),
					listRepos(),
					listServiceMaps(),
				]);
				if (cancelled) {
					return;
				}
				setConfig(cfgRes.data);
				setRepos(reposRes.data);
				setServiceMaps(mapsRes.data);
			} catch {
				// silently ignore load errors; individual save will surface errors
			}
		};

		void load();
		return (): void => {
			cancelled = true;
		};
	}, []);

	return {
		config,
		setConfig,
		repos,
		setRepos,
		serviceMaps,
		setServiceMaps,
	};
}
