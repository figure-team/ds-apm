import React from 'react';
import { useTranslation } from 'react-i18next';
import { Collapse, Form, Input } from 'antd';
import { MarkdownRenderer } from 'components/MarkdownRenderer/MarkdownRenderer';

import { MsTeamsChannel } from '../../CreateAlertChannels/config';

function MsTeams({ setSelectedConfig }: MsTeamsProps): JSX.Element {
	const { t } = useTranslation('channels');

	return (
		<>
			<Form.Item
				name="webhook_url"
				label={t('field_webhook_url')}
				tooltip={{
					title: (
						<MarkdownRenderer
							markdownContent={t('tooltip_ms_teams_url')}
							variables={{}}
						/>
					),
					overlayInnerStyle: { maxWidth: 400 },
					placement: 'right',
				}}
			>
				<Input
					onChange={(event): void => {
						setSelectedConfig((value) => ({
							...value,
							webhook_url: event.target.value,
						}));
					}}
					data-testid="webhook-url-textbox"
				/>
			</Form.Item>

			<Collapse ghost>
				<Collapse.Panel header={t('label_advanced_settings')} key="advanced">
					<Form.Item
						name="title"
						label={t('field_slack_title')}
						help={t('help_channel_title')}
					>
						<Input.TextArea
							rows={4}
							onChange={(event): void =>
								setSelectedConfig((value) => ({
									...value,
									title: event.target.value,
								}))
							}
							data-testid="title-textarea"
						/>
					</Form.Item>

					<Form.Item
						name="text"
						label={t('field_slack_description')}
						help={t('help_channel_description')}
					>
						<Input.TextArea
							rows={8}
							onChange={(event): void =>
								setSelectedConfig((value) => ({
									...value,
									text: event.target.value,
								}))
							}
							data-testid="description-textarea"
							placeholder={t('placeholder_slack_description')}
						/>
					</Form.Item>
				</Collapse.Panel>
			</Collapse>
		</>
	);
}

interface MsTeamsProps {
	setSelectedConfig: React.Dispatch<
		React.SetStateAction<Partial<MsTeamsChannel>>
	>;
}

export default MsTeams;
