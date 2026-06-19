import { ChangeEvent, KeyboardEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useHistory } from 'react-router-dom';
import { Button, DialogWrapper, Input, toast } from '@signozhq/ui';
import { Flex, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import ROUTES from 'constants/routes';
import { ArrowRight, Cable, Check } from 'lucide-react';
import { useAppContext } from 'providers/App/App';
import { routePermission } from 'utils/permission';

import './IntegrationsHeader.styles.scss';

interface IntegrationsHeaderProps {
	searchQuery: string;
	onSearchChange: (value: string) => void;
}

function IntegrationsHeader(props: IntegrationsHeaderProps): JSX.Element {
	const { t } = useTranslation('integrations');
	const history = useHistory();
	const { user } = useAppContext();

	const { searchQuery, onSearchChange } = props;
	const [isRequestIntegrationDialogOpen, setIsRequestIntegrationDialogOpen] =
		useState(false);

	const [
		isSubmittingRequestForIntegration,
		setIsSubmittingRequestForIntegration,
	] = useState(false);

	const [requestedIntegrationName, setRequestedIntegrationName] = useState('');

	const isGetStartedWithCloudAllowed =
		routePermission.GET_STARTED_WITH_CLOUD.includes(user.role);

	const handleRequestIntegrationSubmit = async (): Promise<void> => {
		try {
			setIsSubmittingRequestForIntegration(true);
			const eventName = 'Integration requested';
			const screenName = 'Integration list page';

			const response = await logEvent(eventName, {
				screen: screenName,
				integration: requestedIntegrationName,
			});

			if (response.statusCode === 200) {
				toast.success('Integration Request Submitted', {
					position: 'top-right',
				});
				setRequestedIntegrationName('');
				setIsRequestIntegrationDialogOpen(false);
				setIsSubmittingRequestForIntegration(false);
			} else {
				toast.error(response.error || 'Something went wrong', {
					position: 'top-right',
				});

				setIsSubmittingRequestForIntegration(false);
			}
		} catch (error) {
			toast.error('Something went wrong', {
				position: 'top-right',
			});
			setIsSubmittingRequestForIntegration(false);
		}
	};

	return (
		<div className="integrations-header">
			<Typography.Title className="title">{t('common.integrations')}</Typography.Title>
			<Flex
				justify="space-between"
				align="center"
				className="integrations-header__subrow"
			>
				<Typography.Text className="subtitle">
					{t('header.subtitle')}
				</Typography.Text>
			</Flex>

			<div className="integrations-search-request-container">
				<Input
					placeholder={t('header.search_placeholder')}
					value={searchQuery}
					onChange={(e: ChangeEvent<HTMLInputElement>): void =>
						onSearchChange(e.target.value)
					}
				/>
				<Button
					variant="solid"
					color="secondary"
					className="request-integration-btn"
					prefix={<Cable size={14} />}
					onClick={(): void => setIsRequestIntegrationDialogOpen(true)}
				>
					{t('header.request_integration')}
				</Button>

				<DialogWrapper
					className="request-integration-dialog"
					title={t('header.request_new_integration')}
					open={isRequestIntegrationDialogOpen}
					onOpenChange={setIsRequestIntegrationDialogOpen}
				>
					<div className="request-integration-form">
						<div className="request-integration-form-title">
							{t('header.which_integration')}
						</div>
						<Input
							placeholder={t('header.integration_name_placeholder')}
							value={requestedIntegrationName}
							onChange={(e: ChangeEvent<HTMLInputElement>): void => {
								setRequestedIntegrationName(e.target.value);
							}}
							onKeyDown={(e: KeyboardEvent<HTMLInputElement>): void => {
								if (e.key === 'Enter' && requestedIntegrationName?.trim().length > 0) {
									handleRequestIntegrationSubmit();
								}
							}}
							disabled={isSubmittingRequestForIntegration}
						/>
					</div>

					<div className="request-integration-form-footer">
						<Button
							variant="solid"
							color="primary"
							prefix={<Check size={14} />}
							onClick={handleRequestIntegrationSubmit}
							loading={isSubmittingRequestForIntegration}
							disabled={
								isSubmittingRequestForIntegration ||
								!requestedIntegrationName ||
								requestedIntegrationName?.trim().length === 0
							}
						>
							{t('common.submit')}
						</Button>
					</div>
				</DialogWrapper>

				{isGetStartedWithCloudAllowed && (
					<Button
						variant="solid"
						color="primary"
						onClick={(): void => history.push(ROUTES.GET_STARTED_WITH_CLOUD)}
					>
						<span>{t('header.view_data_sources')}</span>
						<ArrowRight size={14} />
					</Button>
				)}
			</div>
		</div>
	);
}

export default IntegrationsHeader;
