import { render, screen } from 'tests/test-utils';

import LocalizedEmpty from './index';

// 전역 react-i18next mock의 t는 인자를 그대로 반환하므로 키로 검증한다.
describe('LocalizedEmpty', () => {
	it('renders the localized no_data description', () => {
		render(<LocalizedEmpty componentName="Table" />);
		expect(screen.getByText('no_data')).toBeInTheDocument();
	});

	it('renders without a componentName', () => {
		render(<LocalizedEmpty />);
		expect(screen.getByText('no_data')).toBeInTheDocument();
	});
});
