import { Empty } from 'antd';
import { useTranslation } from 'react-i18next';

// antd ko_KR 로케일은 Empty 설명을 정의하지 않아 기본값이 'No data'로 남는다.
// ConfigProvider renderEmpty에 연결해 빈 상태 문구를 전역으로 한글화한다.
// useTranslation('common')으로 네임스페이스 로드와 언어 변경 재렌더를 보장한다.
function LocalizedEmpty({ componentName }: LocalizedEmptyProps): JSX.Element {
	const { t } = useTranslation('common');
	const simple = componentName === 'Table' || componentName === 'List';
	return (
		<Empty
			image={simple ? Empty.PRESENTED_IMAGE_SIMPLE : Empty.PRESENTED_IMAGE_DEFAULT}
			description={t('no_data')}
		/>
	);
}

interface LocalizedEmptyProps {
	componentName?: string;
}

LocalizedEmpty.defaultProps = {
	componentName: undefined,
};

export default LocalizedEmpty;
