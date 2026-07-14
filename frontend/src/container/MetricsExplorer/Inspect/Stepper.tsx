import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Button, Typography } from 'antd';
import classNames from 'classnames';
import { ArrowUpRightFromSquare, RefreshCcw } from 'lucide-react';

import { SPACE_AGGREGATION_LINK, TEMPORAL_AGGREGATION_LINK } from './constants';
import { InspectionStep, StepperProps } from './types';

import './Stepper.styles.scss';

function Stepper({
	inspectionStep,
	resetInspection,
}: StepperProps): JSX.Element {
	const { t } = useTranslation('metricsExplorer');
	return (
		<div className="home-checklist-container">
			<div className="home-checklist-title">
				<Typography.Text>
					{t('welcome_inspector')}
				</Typography.Text>
				<Typography.Text>{t('lets_get_started')}</Typography.Text>
			</div>
			<div className="completed-checklist-container whats-next-checklist-container">
				<div
					className={classNames({
						'completed-checklist-item':
							inspectionStep > InspectionStep.TIME_AGGREGATION,
						'whats-next-checklist-item':
							inspectionStep <= InspectionStep.TIME_AGGREGATION,
					})}
				>
					<div
						className={classNames({
							'completed-checklist-item-title':
								inspectionStep > InspectionStep.TIME_AGGREGATION,
							'whats-next-checklist-item-title':
								inspectionStep <= InspectionStep.TIME_AGGREGATION,
						})}
					>
						{t('first_align_select')}{' '}
						<Typography.Link href={TEMPORAL_AGGREGATION_LINK} target="_blank">
							{t('temporal_aggregation')}{' '}
							<ArrowUpRightFromSquare color={Color.BG_ROBIN_500} size={10} />
						</Typography.Link>
					</div>
				</div>

				<div
					className={classNames({
						'completed-checklist-item':
							inspectionStep > InspectionStep.SPACE_AGGREGATION,
						'whats-next-checklist-item':
							inspectionStep <= InspectionStep.SPACE_AGGREGATION,
					})}
				>
					<div
						className={classNames({
							'completed-checklist-item-title':
								inspectionStep > InspectionStep.SPACE_AGGREGATION,
							'whats-next-checklist-item-title':
								inspectionStep <= InspectionStep.SPACE_AGGREGATION,
						})}
					>
						{t('add_label')}{' '}
						<Typography.Link href={SPACE_AGGREGATION_LINK} target="_blank">
							{t('spatial_aggregation')}{' '}
							<ArrowUpRightFromSquare color={Color.BG_ROBIN_500} size={10} />
						</Typography.Link>
					</div>
				</div>
			</div>

			<div className="completed-message-container">
				{inspectionStep === InspectionStep.COMPLETED && (
					<>
						<Typography.Text>
							{t('tutorial_completed')}
						</Typography.Text>
						<Typography.Text>
							{t('inspect_new_or_reset')}
						</Typography.Text>
						<Button icon={<RefreshCcw size={12} />} onClick={resetInspection}>
							{t('reset_query')}
						</Button>
					</>
				)}
			</div>
		</div>
	);
}

export default Stepper;
