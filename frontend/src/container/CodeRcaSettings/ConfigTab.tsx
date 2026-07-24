import RcaConfigCard from './components/RcaConfigCard';
import RcaReposCard from './components/RcaReposCard';
import RcaServiceMapsCard from './components/RcaServiceMapsCard';
import { useCodeRcaSettingsData } from './hooks/useCodeRcaSettingsData';

interface Props {
	isAdmin: boolean;
}

/** Code RCA 설정 탭 — 데이터 로드는 훅에, 도메인별 UI는 카드 3개에 위임한다. */
function ConfigTab({ isAdmin }: Props): JSX.Element {
	const { config, setConfig, repos, setRepos, serviceMaps, setServiceMaps } =
		useCodeRcaSettingsData();

	return (
		<div>
			<RcaConfigCard config={config} setConfig={setConfig} isAdmin={isAdmin} />
			<RcaReposCard repos={repos} setRepos={setRepos} isAdmin={isAdmin} />
			<RcaServiceMapsCard
				repos={repos}
				serviceMaps={serviceMaps}
				setServiceMaps={setServiceMaps}
				isAdmin={isAdmin}
			/>
		</div>
	);
}

export default ConfigTab;
