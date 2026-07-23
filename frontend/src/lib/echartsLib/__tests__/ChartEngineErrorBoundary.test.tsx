import { render, screen } from '@testing-library/react';

import ChartEngineErrorBoundary from '../components/ChartEngineErrorBoundary';

function Bomb(): JSX.Element {
	throw new Error('render crash');
}

describe('ChartEngineErrorBoundary', () => {
	it('자식 렌더 예외 시 onError 호출 + fallback 렌더', () => {
		const onError = jest.fn();
		// React ErrorBoundary 테스트의 console.error 노이즈 억제
		jest.spyOn(console, 'error').mockImplementation(() => undefined);
		render(
			<ChartEngineErrorBoundary
				onError={onError}
				fallback={<div data-testid="uplot-fallback" />}
			>
				<Bomb />
			</ChartEngineErrorBoundary>,
		);
		expect(onError).toHaveBeenCalled();
		expect(screen.getByTestId('uplot-fallback')).toBeInTheDocument();
		(console.error as jest.Mock).mockRestore();
	});
});
