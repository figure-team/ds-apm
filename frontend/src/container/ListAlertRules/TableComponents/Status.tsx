import { Tag } from 'antd';
import type { RuletypesRuleDTO } from 'api/generated/services/sigNoz.schemas';
import { useIsDarkMode } from 'hooks/useDarkMode';

function Status({ status }: StatusProps): JSX.Element {
	const isDarkMode = useIsDarkMode();

	switch (status) {
		case 'inactive': {
			return <Tag color={isDarkMode ? 'green' : '#16A34A'}>OK</Tag>;
		}

		case 'pending': {
			return <Tag color={isDarkMode ? 'orange' : '#F59E0B'}>Pending</Tag>;
		}

		case 'firing': {
			return <Tag color={isDarkMode ? 'red' : '#DC2626'}>Firing</Tag>;
		}

		case 'disabled': {
			return <Tag>Disabled</Tag>;
		}

		default: {
			return <Tag color="default">Unknown</Tag>;
		}
	}
}

interface StatusProps {
	status: RuletypesRuleDTO['state'];
}

export default Status;
