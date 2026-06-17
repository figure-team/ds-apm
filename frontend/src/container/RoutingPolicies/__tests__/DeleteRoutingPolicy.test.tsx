import { fireEvent, screen } from '@testing-library/react';
import { render } from 'tests/test-utils';

import DeleteRoutingPolicy from '../DeleteRoutingPolicy';
import { MOCK_ROUTING_POLICY_1 } from './testUtils';

const mockRoutingPolicy = MOCK_ROUTING_POLICY_1;
const mockHandleDelete = jest.fn();
const mockHandleClose = jest.fn();

const DELETE_DIALOG_TITLE = 'rp_delete_title';
const DELETE_BUTTON_TEXT = 'rp_delete_confirm';
const CANCEL_BUTTON_TEXT = 'rp_delete_cancel';

describe('DeleteRoutingPolicy', () => {
	it('renders base layout with routing policy', () => {
		render(
			<DeleteRoutingPolicy
				routingPolicy={mockRoutingPolicy}
				isDeletingRoutingPolicy={false}
				handleDelete={mockHandleDelete}
				handleClose={mockHandleClose}
			/>,
		);
		expect(
			screen.getByRole('dialog', { name: DELETE_DIALOG_TITLE }),
		).toBeInTheDocument();
		expect(screen.getByText('alerts:rp_delete_text')).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: CANCEL_BUTTON_TEXT }),
		).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: DELETE_BUTTON_TEXT }),
		).toBeInTheDocument();
	});

	it('should call handleDelete when delete button is clicked', () => {
		render(
			<DeleteRoutingPolicy
				routingPolicy={mockRoutingPolicy}
				isDeletingRoutingPolicy={false}
				handleDelete={mockHandleDelete}
				handleClose={mockHandleClose}
			/>,
		);
		fireEvent.click(screen.getByRole('button', { name: DELETE_BUTTON_TEXT }));
		expect(mockHandleDelete).toHaveBeenCalled();
	});

	it('should call handleClose when cancel button is clicked', () => {
		render(
			<DeleteRoutingPolicy
				routingPolicy={mockRoutingPolicy}
				isDeletingRoutingPolicy={false}
				handleDelete={mockHandleDelete}
				handleClose={mockHandleClose}
			/>,
		);
		fireEvent.click(screen.getByRole('button', { name: CANCEL_BUTTON_TEXT }));
		expect(mockHandleClose).toHaveBeenCalled();
	});

	it('should be disabled when deleting routing policy', () => {
		render(
			<DeleteRoutingPolicy
				routingPolicy={mockRoutingPolicy}
				isDeletingRoutingPolicy
				handleDelete={mockHandleDelete}
				handleClose={mockHandleClose}
			/>,
		);
		expect(
			screen.getByRole('button', { name: DELETE_BUTTON_TEXT }),
		).toBeDisabled();
		expect(
			screen.getByRole('button', { name: CANCEL_BUTTON_TEXT }),
		).toBeDisabled();
	});
});
