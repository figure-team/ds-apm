import { render } from '@testing-library/react';

import EChartsView from '../components/EChartsView';
import mockedEchartsCore, { EChartsType } from '../echartsCore';

interface MockChartInstance {
	setOption: jest.Mock;
	resize: jest.Mock;
	dispose: jest.Mock;
	on: jest.Mock;
	off: jest.Mock;
}

function createMockInstance(): MockChartInstance {
	return {
		setOption: jest.fn(),
		resize: jest.fn(),
		dispose: jest.fn(),
		on: jest.fn(),
		off: jest.fn(),
	};
}

// mock 인스턴스는 EChartsType의 극히 일부 메서드만 흉내내므로, echarts.init의
// 실제 반환 타입(EChartsType)과의 경계에서만 캐스팅한다 — 테스트 본문에서는
// MockChartInstance(jest.fn 메서드)로 다뤄야 mockClear 등을 그대로 쓸 수 있다
function asEChartsType(instance: MockChartInstance): EChartsType {
	return (instance as unknown) as EChartsType;
}

jest.mock('../echartsCore', () => ({
	__esModule: true,
	default: {
		init: jest.fn(),
		registerTheme: jest.fn(),
	},
}));

const mockInit = jest.mocked(mockedEchartsCore.init);

// init 호출 순번으로 그 호출이 반환한 인스턴스를 조회한다 (호출마다 새 객체이므로
// 구인스턴스/신인스턴스를 구분해 재생성 시퀀스·불필요 재생성 방지를 검증할 수 있다)
function getInstance(callIndex: number): MockChartInstance {
	return (mockInit.mock.results[callIndex]
		.value as unknown) as MockChartInstance;
}

describe('EChartsView', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		mockInit.mockImplementation(() => asEChartsType(createMockInstance()));
	});

	const baseProps = {
		option: { series: [] },
		width: 400,
		height: 300,
		isDarkMode: true,
		onError: jest.fn(),
	};

	it('마운트 시 init 후 replaceMerge로 setOption한다', () => {
		render(<EChartsView {...baseProps} />);
		const instance = getInstance(0);
		expect(instance.setOption).toHaveBeenCalledWith(
			baseProps.option,
			expect.objectContaining({ replaceMerge: ['series'] }),
		);
	});

	it('width/height 변경 시 무애니메이션 resize', () => {
		const { rerender } = render(<EChartsView {...baseProps} />);
		const instance = getInstance(0);
		rerender(<EChartsView {...baseProps} width={500} />);
		expect(instance.resize).toHaveBeenCalledWith(
			expect.objectContaining({ animation: { duration: 0 } }),
		);
	});

	it('언마운트 시 dispose한다', () => {
		const { unmount } = render(<EChartsView {...baseProps} />);
		const instance = getInstance(0);
		unmount();
		expect(instance.dispose).toHaveBeenCalled();
	});

	it('setOption 예외는 onError로 전달된다 (fail-open)', () => {
		mockInit.mockImplementationOnce(() => {
			const instance = createMockInstance();
			instance.setOption.mockImplementationOnce(() => {
				throw new Error('bad option');
			});
			return asEChartsType(instance);
		});
		const onError = jest.fn();
		render(<EChartsView {...baseProps} onError={onError} />);
		expect(onError).toHaveBeenCalledWith(expect.any(Error));
	});

	it('init 예외는 onError로 전달된다 (fail-open)', () => {
		mockInit.mockImplementationOnce(() => {
			throw new Error('init failed');
		});
		const onError = jest.fn();
		render(<EChartsView {...baseProps} onError={onError} />);
		expect(onError).toHaveBeenCalledWith(expect.any(Error));
	});

	it('resize 예외는 onError로 전달된다 (fail-open)', () => {
		const onError = jest.fn();
		const { rerender } = render(
			<EChartsView {...baseProps} onError={onError} />,
		);
		const instance = getInstance(0);
		instance.resize.mockImplementationOnce(() => {
			throw new Error('resize failed');
		});
		rerender(<EChartsView {...baseProps} onError={onError} width={500} />);
		expect(onError).toHaveBeenCalledWith(expect.any(Error));
	});

	it('마운트 시 onInstanceReady가 신규 인스턴스로 1회 호출된다', () => {
		const onInstanceReady = jest.fn();
		render(
			<EChartsView {...baseProps} onInstanceReady={onInstanceReady} />,
		);
		const instance = getInstance(0);
		expect(onInstanceReady).toHaveBeenCalledTimes(1);
		expect(onInstanceReady).toHaveBeenCalledWith(instance);
	});

	it('isDarkMode 변경 시 dispose→init→setOption 순으로 인스턴스를 재생성한다', () => {
		const callOrder: string[] = [];
		mockInit.mockImplementation(() => {
			callOrder.push('init');
			const instance = createMockInstance();
			instance.dispose.mockImplementation(() => {
				callOrder.push('dispose');
			});
			instance.setOption.mockImplementation(() => {
				callOrder.push('setOption');
			});
			return asEChartsType(instance);
		});

		const { rerender } = render(<EChartsView {...baseProps} />);
		const oldInstance = getInstance(0);
		callOrder.length = 0; // 마운트 시점 init/setOption 로그는 제외하고 재생성 시퀀스만 검증
		oldInstance.setOption.mockClear(); // 마운트 시 1회 호출된 이력을 지워 재호출 여부만 검증

		rerender(<EChartsView {...baseProps} isDarkMode={false} />);

		expect(mockInit).toHaveBeenCalledTimes(2);
		const newInstance = getInstance(1);
		expect(newInstance).not.toBe(oldInstance);
		expect(oldInstance.dispose).toHaveBeenCalledTimes(1);
		expect(newInstance.setOption).toHaveBeenCalledWith(
			baseProps.option,
			expect.objectContaining({ replaceMerge: ['series'] }),
		);
		// 구인스턴스는 버려지므로 재생성 시퀀스 동안 setOption이 다시 호출되지 않는다
		expect(oldInstance.setOption).not.toHaveBeenCalled();
		expect(callOrder).toEqual(['dispose', 'init', 'setOption']);
	});

	it('option 새 객체·onError/onInstanceReady 새 참조만으로는 인스턴스를 재생성하지 않는다', () => {
		const { rerender } = render(<EChartsView {...baseProps} />);
		const instance = getInstance(0);
		expect(mockInit).toHaveBeenCalledTimes(1);
		instance.setOption.mockClear();

		const newOption = { series: [] }; // 매 렌더 새로 만들어진 객체 참조
		rerender(
			<EChartsView
				{...baseProps}
				option={newOption}
				onError={jest.fn()}
				onInstanceReady={jest.fn()}
			/>,
		);

		expect(mockInit).toHaveBeenCalledTimes(1); // 재생성(init 재호출) 없음
		expect(instance.setOption).toHaveBeenCalledWith(
			newOption,
			expect.objectContaining({ replaceMerge: ['series'] }),
		);
	});
});
