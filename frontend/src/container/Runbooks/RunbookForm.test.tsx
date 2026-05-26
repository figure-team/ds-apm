import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import RunbookForm from './RunbookForm';

beforeEach(() => jest.clearAllMocks());

it('submits the form values on save', async () => {
	const onSubmit = jest.fn();
	render(<RunbookForm onSubmit={onSubmit} onCancel={jest.fn()} />);
	fireEvent.change(screen.getByLabelText(/title/i), { target: { value: 'My runbook' } });
	fireEvent.change(screen.getByLabelText(/script \(bash\)/i), { target: { value: '#!/bin/bash\nhi\n' } });
	fireEvent.click(screen.getByRole('button', { name: /save/i }));
	await waitFor(() => expect(onSubmit).toHaveBeenCalled());
	const submitted = onSubmit.mock.calls[0][0];
	expect(submitted.title).toBe('My runbook');
	expect(submitted.executableScript).toContain('#!/bin/bash');
});

it('prefills initial values when editing', () => {
	render(
		<RunbookForm
			initial={{
				title: 'Existing title',
				description: 'Existing description',
				executableScript: '#!/bin/bash\necho original\n',
				status: 'approved',
			}}
			onSubmit={jest.fn()}
			onCancel={jest.fn()}
		/>,
	);
	expect(screen.getByDisplayValue('Existing title')).toBeInTheDocument();
	expect(screen.getByDisplayValue(/#!\/bin\/bash/)).toBeInTheDocument();
});
