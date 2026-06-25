import { fireEvent, render, screen, waitFor } from 'tests/test-utils';

import RunbookForm from './RunbookForm';

beforeEach(() => jest.clearAllMocks());

it('submits the form values on save', async () => {
	const onSubmit = jest.fn();
	render(<RunbookForm onSubmit={onSubmit} onCancel={jest.fn()} />);
	// i18n mock returns the key, so labels/buttons assert on their translation keys.
	fireEvent.change(screen.getByLabelText('field_title'), { target: { value: 'My runbook' } });
	fireEvent.change(screen.getByLabelText('field_script'), { target: { value: '#!/bin/bash\nhi\n' } });
	fireEvent.click(screen.getByRole('button', { name: 'btn_save' }));
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
