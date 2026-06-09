import { Button, Typography } from 'antd';
import Modal from 'components/Modal';
import { useTranslation } from 'react-i18next';

function SkipOnBoardingModal({ onContinueClick }: Props): JSX.Element {
	const { t } = useTranslation(['services']);
	return (
		<Modal
			title={t('setup_instrumentation')}
			isModalVisible
			closable={false}
			footer={[
				<Button key="submit" type="primary" onClick={onContinueClick}>
					{t('continue_without_instrumentation')}
				</Button>,
			]}
		>
			<>
				<iframe
					width="100%"
					height="265"
					src="https://www.youtube.com/embed/J1Bof55DOb4"
					frameBorder="0"
					allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
					allowFullScreen
					title="youtube_video"
				/>
				<div>
					<Typography>{t('no_instrumentation_data')}</Typography>
					<Typography>
						{t('instrument_your_application')}&nbsp;
						<a
							href="https://signoz.io/docs/instrumentation/overview"
							target="_blank"
							rel="noreferrer"
						>
							{t('here')}
						</a>
					</Typography>
				</div>
			</>
		</Modal>
	);
}

interface Props {
	onContinueClick: () => void;
}

export default SkipOnBoardingModal;
