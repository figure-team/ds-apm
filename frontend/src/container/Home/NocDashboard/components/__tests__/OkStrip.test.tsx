import { render, screen } from '@testing-library/react';

import OkStrip from '../OkStrip';

jest.mock('react-i18next', () => ({
	useTranslation: () => ({ t: (k: string) => k }),
}));
jest.mock('hooks/useSafeNavigate', () => ({
	useSafeNavigate: () => ({ safeNavigate: jest.fn() }),
}));

describe('OkStrip', () => {
	it('renders chips and a +N overflow chip', () => {
		render(<OkStrip names={['a', 'b', 'c', 'd']} maxChips={2} />);
		expect(screen.getByText('a')).toBeInTheDocument();
		expect(screen.getByText('b')).toBeInTheDocument();
		expect(screen.getByText('+2')).toBeInTheDocument(); // c,d collapsed
	});

	it('renders nothing extra when within limit', () => {
		render(<OkStrip names={['a']} maxChips={8} />);
		expect(screen.queryByText(/^\+/)).not.toBeInTheDocument();
	});
});
