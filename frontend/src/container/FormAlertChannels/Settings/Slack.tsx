import { Dispatch, SetStateAction } from 'react';
import { useTranslation } from 'react-i18next';
import { Collapse, Form, Input } from 'antd';
import { MarkdownRenderer } from 'components/MarkdownRenderer/MarkdownRenderer';

import { SlackChannel } from '../../CreateAlertChannels/config';

const { TextArea } = Input;

function Slack({ setSelectedConfig }: SlackProps): JSX.Element {
	const { t } = useTranslation('channels');

	return (
		<>
			<Form.Item
				name="api_url"
				label={t('field_webhook_url')}
				tooltip={{
					title: (
						<MarkdownRenderer
							markdownContent={t('tooltip_slack_url')}
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
							api_url: event.target.value,
						}));
					}}
					data-testid="webhook-url-textbox"
				/>
			</Form.Item>

			<Form.Item
				name="channel"
				help={t('slack_channel_help')}
				label={t('field_slack_recipient')}
			>
				<Input
					onChange={(event): void =>
						setSelectedConfig((value) => ({
							...value,
							channel: event.target.value,
						}))
					}
					data-testid="slack-channel-textbox"
				/>
			</Form.Item>

			<Collapse ghost>
				<Collapse.Panel header={t('label_advanced_settings')} key="advanced">
					<Form.Item
						name="title"
						label={t('field_slack_title')}
						help={t('help_channel_title')}
					>
						<TextArea
							data-testid="title-textarea"
							rows={4}
							onChange={(event): void =>
								setSelectedConfig((value) => ({
									...value,
									title: event.target.value,
								}))
							}
						/>
					</Form.Item>

					<Form.Item
						name="text"
						label={t('field_slack_description')}
						help={t('help_channel_description')}
					>
						<TextArea
							rows={8}
							onChange={(event): void =>
								setSelectedConfig((value) => ({
									...value,
									text: event.target.value,
								}))
							}
							placeholder={t('placeholder_slack_description')}
							data-testid="description-textarea"
						/>
					</Form.Item>
				</Collapse.Panel>
			</Collapse>
		</>
	);
}

interface SlackProps {
	setSelectedConfig: Dispatch<SetStateAction<Partial<SlackChannel>>>;
}

export default Slack;
