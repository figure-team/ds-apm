import { useTranslation } from 'react-i18next';
import { UseQueryResult } from 'react-query';
import { SuccessResponse } from 'types/api';

import noDataUrl from '@/assets/Icons/no-data.svg';

import EndPointsDropDown from './EndPointsDropDown';

function EndPointDetailsZeroState({
	setSelectedEndPointName,
	endPointDropDownDataQuery,
}: {
	setSelectedEndPointName: (endPointName: string) => void;
	endPointDropDownDataQuery: UseQueryResult<SuccessResponse<any>>;
}): JSX.Element {
	const { t } = useTranslation('apiMonitoring');
	return (
		<div className="end-point-details-zero-state-wrapper">
			<div className="end-point-details-zero-state-content">
				<img
					src={noDataUrl}
					alt="no-data"
					width={32}
					height={32}
					className="end-point-details-zero-state-icon"
				/>
				<div className="end-point-details-zero-state-content-wrapper">
					<div className="end-point-details-zero-state-text-content">
						<div className="title">{t('no_endpoint_selected')}</div>
						<div className="description">{t('select_endpoint_to_see')}</div>
					</div>
					<EndPointsDropDown
						setSelectedEndPointName={setSelectedEndPointName}
						endPointDropDownDataQuery={endPointDropDownDataQuery}
						parentContainerDiv=".end-point-details-zero-state-wrapper"
						dropdownStyle={{ width: '60%' }}
					/>
				</div>
			</div>
		</div>
	);
}

export default EndPointDetailsZeroState;
