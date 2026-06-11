import { useTranslation } from 'react-i18next';
import { Typography } from 'antd';
import eyesEmojiUrl from 'assets/Images/eyesEmoji.svg';

import styles from './QueryCancelledPlaceholder.module.scss';

interface QueryCancelledPlaceholderProps {
	subText?: string;
}

function QueryCancelledPlaceholder({
	subText,
}: QueryCancelledPlaceholderProps): JSX.Element {
	const { t } = useTranslation('common');
	return (
		<div className={styles.placeholder}>
			<img className={styles.emoji} src={eyesEmojiUrl} alt={t('eyes_emoji_alt')} />
			<Typography className={styles.text}>
				{t('query_cancelled')}
				<span className={styles.subText}>
					{' '}
					{subText || t('click_run_query_data')}
				</span>
			</Typography>
		</div>
	);
}

QueryCancelledPlaceholder.defaultProps = {
	subText: undefined,
};

export default QueryCancelledPlaceholder;
