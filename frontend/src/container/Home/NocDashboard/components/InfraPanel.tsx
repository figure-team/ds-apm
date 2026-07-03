import { NocInfraHost } from '../types';

export interface InfraPanelProps {
	hosts: NocInfraHost[];
	isLoading: boolean;
	isError: boolean;
}

// SEED STUB — 본문은 Lane D가 채움(impl-plan Task 10). props 인터페이스는 계약(불변).
export default function InfraPanel(_props: InfraPanelProps): JSX.Element {
	return <div className="noc-c2-infra" data-stub="infra" />;
}
