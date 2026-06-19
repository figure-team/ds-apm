import React, { useCallback, useState } from 'react';
import { PlusOutlined } from '@ant-design/icons';
import { Button, Divider, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import ROUTES from 'constants/routes';
import useComponentPermission from 'hooks/useComponentPermission';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import { useAppContext } from 'providers/App/App';
import { useTranslation } from 'react-i18next';
import { DataSource } from 'types/common/queryBuilder';
import { isModifierKeyPressed } from 'utils/app';

import alertEmojiUrl from '@/assets/Icons/alert_emoji.svg';

import AlertInfoCard from './AlertInfoCard';
import { ALERT_CARDS, ALERT_INFO_LINKS } from './alertLinks';
import InfoLinkText from './InfoLinkText';

import './AlertsEmptyState.styles.scss';

const alertLogEvents = (
	title: string,
	link: string,
	dataSource?: DataSource,
): void => {
	const attributes = {
		link,
		page: 'Alert empty state page',
	};

	logEvent(title, dataSource ? { ...attributes, dataSource } : attributes);
};

export function AlertsEmptyState(): JSX.Element {
	const { t } = useTranslation('alerts');
	const { user } = useAppContext();
	const { safeNavigate } = useSafeNavigate();
	const [addNewAlert] = useComponentPermission(
		['add_new_alert', 'action'],
		user.role,
	);

	const [loading, setLoading] = useState(false);

	const onClickNewAlertHandler = useCallback(
		(e: React.MouseEvent) => {
			setLoading(false);
			safeNavigate(ROUTES.ALERTS_NEW, { newTab: isModifierKeyPressed(e) });
		},
		[safeNavigate],
	);

	return (
		<div className="alert-list-container">
			<div className="alert-list-view-content">
				<div className="alert-list-title-container">
					<Typography.Title className="title">{t('list_alert_rules_title')}</Typography.Title>
					<Typography.Text className="subtitle">
						{t('list_alert_rules_subtitle')}
					</Typography.Text>
				</div>
				<section className="empty-alert-info-container">
					<div className="alert-content">
						<section className="heading">
							<img
								src={alertEmojiUrl}
								alt="alert-header"
								style={{ height: '32px', width: '32px' }}
							/>
							<div>
								<Typography.Text className="empty-info">
									{t('list_no_alert_rules')}{' '}
								</Typography.Text>
								<Typography.Text className="empty-alert-action">
									{t('list_create_alert_rule')}
								</Typography.Text>
							</div>
						</section>
						<div className="action-container">
							<Button
								className="add-alert-btn"
								onClick={onClickNewAlertHandler}
								icon={<PlusOutlined />}
								disabled={!addNewAlert}
								loading={loading}
								type="primary"
								data-testid="add-alert"
							>
								{t('list_new_alert_rule')}
							</Button>
							<InfoLinkText
								infoText="Watch a tutorial on creating a sample alert"
								link="https://youtu.be/xjxNIqiv4_M"
								leftIconVisible
								rightIconVisible
								onClick={(): void =>
									alertLogEvents(
										'Alert: Video tutorial link clicked',
										'https://youtu.be/xjxNIqiv4_M',
									)
								}
							/>
						</div>

						{ALERT_INFO_LINKS.map((info) => {
							const logEventTriggered = (): void =>
								alertLogEvents(
									'Alert: Tutorial doc link clicked',
									info.link,
									info.dataSource,
								);
							return (
								<InfoLinkText
									key={info.link}
									infoText={info.infoText}
									link={info.link}
									leftIconVisible={info.leftIconVisible}
									rightIconVisible={info.rightIconVisible}
									onClick={logEventTriggered}
								/>
							);
						})}
					</div>
				</section>
				<div className="get-started-text">
					<Divider>
						<Typography.Text className="get-started-text">
							{t('list_sample_alerts')}
						</Typography.Text>
					</Divider>
				</div>

				{ALERT_CARDS.map((card) => {
					const logEventTriggered = (): void =>
						alertLogEvents(
							'Alert: Sample alert link clicked',
							card.link,
							card.dataSource,
						);
					return (
						<AlertInfoCard
							key={card.link}
							header={card.header}
							subheader={card.subheader}
							link={card.link}
							onClick={logEventTriggered}
						/>
					);
				})}
			</div>
		</div>
	);
}
