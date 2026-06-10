import { InputNumber, Row, Space, Typography } from 'antd';
import { useTranslation } from 'react-i18next';

interface PopoverContentProps {
	linesPerRow: number;
	handleLinesPerRowChange: (l: unknown) => void;
}

function PopoverContent({
	linesPerRow,
	handleLinesPerRowChange,
}: PopoverContentProps): JSX.Element {
	const { t } = useTranslation(['logs']);
	return (
		<Row align="middle">
			<Space align="center">
				<Typography>{t('max_lines_per_row')} </Typography>
				<InputNumber
					min={1}
					max={10}
					value={linesPerRow}
					onChange={handleLinesPerRowChange}
				/>
			</Space>
		</Row>
	);
}

export default PopoverContent;
