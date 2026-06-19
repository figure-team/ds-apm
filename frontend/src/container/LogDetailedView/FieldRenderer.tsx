import { useTranslation } from 'react-i18next';
import { Divider, Tooltip, Typography } from 'antd';

import { TagContainer, TagLabel, TagValue } from './FieldRenderer.styles';
import { FieldRendererProps } from './LogDetailedView.types';
import { getFieldAttributes } from './utils';

import './FieldRenderer.styles.scss';

function FieldRenderer({ field }: FieldRendererProps): JSX.Element {
	const { t } = useTranslation(['logs']);
	const { dataType, newField, logType } = getFieldAttributes(field);

	return (
		<span className="field-renderer-container">
			{dataType && newField && logType ? (
				<>
					<Tooltip placement="left" title={newField} mouseLeaveDelay={0}>
						<Typography.Text ellipsis className="label">
							{newField}{' '}
						</Typography.Text>
					</Tooltip>

					<div className="tags">
						<TagContainer>
							<TagLabel>
								{t('logs:field_type')}
								<Divider type="vertical" />{' '}
							</TagLabel>
							<TagValue>{logType}</TagValue>
						</TagContainer>
						<TagContainer>
							<TagLabel>
								{t('logs:field_data_type')} <Divider type="vertical" />{' '}
							</TagLabel>
							<TagValue>{dataType}</TagValue>
						</TagContainer>
					</div>
				</>
			) : (
				<span className="label">{field}</span>
			)}
		</span>
	);
}

export default FieldRenderer;
