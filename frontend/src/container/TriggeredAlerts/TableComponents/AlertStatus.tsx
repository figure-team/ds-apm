import { Tag } from 'antd';
import { useIsDarkMode } from 'hooks/useDarkMode';

function Severity({ severity }: SeverityProps): JSX.Element {
	const isDarkMode = useIsDarkMode();

	switch (severity) {
		case 'unprocessed': {
			return <Tag color={isDarkMode ? 'green' : '#16A34A'}>UnProcessed</Tag>;
		}

		case 'active': {
			return <Tag color={isDarkMode ? 'red' : '#DC2626'}>Firing</Tag>;
		}

		case 'suppressed': {
			return <Tag color={isDarkMode ? 'red' : '#DC2626'}>Suppressed</Tag>;
		}

		default: {
			return <Tag color="default">Unknown Status</Tag>;
		}
	}
}

interface SeverityProps {
	severity: string;
}

export default Severity;
