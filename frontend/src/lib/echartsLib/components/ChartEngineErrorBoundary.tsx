import { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
	onError: (error: unknown) => void;
	fallback: ReactNode;
	children: ReactNode;
}

interface State {
	hasError: boolean;
}

/**
 * ECharts 경로 렌더 예외 → uPlot 폴백 통지 (스펙 §6, fail-open).
 * 명령형 호출 예외는 EChartsView의 try/catch가 담당한다.
 */
export default class ChartEngineErrorBoundary extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.state = { hasError: false };
	}

	static getDerivedStateFromError(): State {
		return { hasError: true };
	}

	componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
		// eslint-disable-next-line no-console
		console.warn('[echartsLib] ECharts 렌더 실패 — uPlot 폴백', error, errorInfo);
		this.props.onError(error);
	}

	render(): ReactNode {
		const { hasError } = this.state;
		const { fallback, children } = this.props;
		return hasError ? fallback : children;
	}
}
