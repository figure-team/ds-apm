import React from 'react';
import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Button } from 'antd';
import { Binoculars, DraftingCompass, ScrollText } from 'lucide-react';

import './GraphControlsPanel.styles.scss';

interface GraphControlsPanelProps {
	id: string;
	onViewLogsClick?: (e: React.MouseEvent) => void;
	onViewTracesClick: (e: React.MouseEvent) => void;
	onViewAPIMonitoringClick?: (e: React.MouseEvent) => void;
}

function GraphControlsPanel({
	id,
	onViewLogsClick,
	onViewTracesClick,
	onViewAPIMonitoringClick,
}: GraphControlsPanelProps): JSX.Element {
	const { t } = useTranslation(['services']);
	return (
		<div id={id} className="graph-controls-panel">
			<Button
				type="link"
				icon={<DraftingCompass size={14} />}
				size="small"
				onClick={onViewTracesClick}
				style={{ color: Color.BG_VANILLA_100 }}
			>
				{t('services:view_traces')}
			</Button>
			{onViewLogsClick && (
				<Button
					type="link"
					icon={<ScrollText size={14} />}
					size="small"
					onClick={onViewLogsClick}
					style={{ color: Color.BG_VANILLA_100 }}
				>
					{t('services:view_logs')}
				</Button>
			)}
			{onViewAPIMonitoringClick && (
				<Button
					type="link"
					icon={<Binoculars size={14} />}
					size="small"
					onClick={onViewAPIMonitoringClick}
					style={{ color: Color.BG_VANILLA_100 }}
				>
					{t('services:view_external_apis')}
				</Button>
			)}
		</div>
	);
}

GraphControlsPanel.defaultProps = {
	onViewLogsClick: undefined,
	onViewAPIMonitoringClick: undefined,
};

export default GraphControlsPanel;
