import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Tooltip, Typography } from 'antd';
import WarningPopover from 'components/WarningPopover/WarningPopover';
import { PANEL_TYPES } from 'constants/queryBuilder';
import { useQueryBuilder } from 'hooks/queryBuilder/useQueryBuilder';
import { DraftingCompass, Plus, Sigma } from 'lucide-react';
import BetaTag from 'periscope/components/BetaTag/BetaTag';

import './QueryFooter.styles.scss';

function TraceOperatorSection({
	addTraceOperator,
}: {
	addTraceOperator?: () => void;
}): JSX.Element {
	const { t } = useTranslation('common');
	const { currentQuery, panelType } = useQueryBuilder();

	const showTraceOperatorWarning = useMemo(() => {
		const isListViewPanel =
			panelType === PANEL_TYPES.LIST || panelType === PANEL_TYPES.TRACE;
		const hasMultipleQueries = currentQuery.builder.queryData.length > 1;
		const hasTraceOperator =
			currentQuery.builder.queryTraceOperator &&
			currentQuery.builder.queryTraceOperator.length > 0;
		return isListViewPanel && hasMultipleQueries && !hasTraceOperator;
	}, [
		currentQuery?.builder?.queryData,
		currentQuery?.builder?.queryTraceOperator,
		panelType,
	]);

	const traceOperatorWarning = useMemo(() => {
		if (currentQuery.builder.queryData.length === 0) {
			return '';
		}
		const firstQuery = currentQuery.builder.queryData[0];
		return t('query_builder.trace_operator_warning', {
			queryName: firstQuery.queryName,
		});
	}, [currentQuery, t]);
	return (
		<div className="qb-trace-operator-button-container">
			<Tooltip
				title={
					<div style={{ textAlign: 'center' }}>
						{t('query_builder.add_trace_matching')}
						<Typography.Link
							href="https://signoz.io/docs/userguide/query-builder-v5/#multi-query-analysis-trace-operators"
							target="_blank"
							style={{ textDecoration: 'underline' }}
						>
							{' '}
							<br />
							{t('learn_more')}
						</Typography.Link>
					</div>
				}
			>
				<Button
					className="add-trace-operator-button periscope-btn"
					icon={<DraftingCompass size={16} />}
					onClick={(): void => addTraceOperator?.()}
				>
					<div className="qb-trace-operator-button-container-text">
						{t('query_builder.add_trace_matching')}
						<BetaTag />
					</div>
				</Button>
			</Tooltip>
			{showTraceOperatorWarning && (
				<WarningPopover message={traceOperatorWarning} />
			)}
		</div>
	);
}

export default function QueryFooter({
	addNewBuilderQuery,
	addNewFormula,
	addTraceOperator,
	showAddFormula = true,
	showAddTraceOperator = false,
}: {
	addNewBuilderQuery: () => void;
	addNewFormula: () => void;
	addTraceOperator?: () => void;
	showAddTraceOperator: boolean;
	showAddFormula?: boolean;
}): JSX.Element {
	const { t } = useTranslation('common');
	return (
		<div className="qb-footer">
			<div className="qb-footer-container">
				<div className="qb-add-new-query">
					<Tooltip
						title={
							<div style={{ textAlign: 'center' }}>
								{t('query_builder.add_new_query')}
							</div>
						}
					>
						<Button
							className="add-new-query-button periscope-btn "
							icon={<Plus size={16} />}
							onClick={addNewBuilderQuery}
						/>
					</Tooltip>
				</div>

				{showAddFormula && (
					<div className="qb-add-formula">
						<Tooltip
							title={
								<div style={{ textAlign: 'center' }}>
									{t('query_builder.add_new_formula')}
									<Typography.Link
										href="https://signoz.io/docs/userguide/query-builder-v5/#multi-query-analysis-advanced-comparisons"
										target="_blank"
										style={{ textDecoration: 'underline' }}
									>
										{' '}
										<br />
										{t('learn_more')}
									</Typography.Link>
								</div>
							}
						>
							<Button
								className="add-formula-button periscope-btn "
								icon={<Sigma size={16} />}
								onClick={addNewFormula}
							>
								{t('query_builder.add_formula')}
							</Button>
						</Tooltip>
					</div>
				)}
				{showAddTraceOperator && (
					<TraceOperatorSection addTraceOperator={addTraceOperator} />
				)}
			</div>
		</div>
	);
}
