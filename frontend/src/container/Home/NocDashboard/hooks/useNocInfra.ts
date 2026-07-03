import { NocInfraHost } from '../types';

export interface UseNocInfraResult {
	hosts: NocInfraHost[];
	isLoading: boolean;
	isError: boolean;
}

// SEED STUB — 본문은 Lane A가 채움(impl-plan Task 3). 시그니처·반환 타입은 계약(불변).
export default function useNocInfra(): UseNocInfraResult {
	return { hosts: [], isLoading: false, isError: false };
}
