import React from 'react';
import { Color } from '@signozhq/design-tokens';
import { Compass, Dot } from '@signozhq/icons';
import logEvent from 'api/common/logEvent';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import history from 'lib/history';
import Card from 'periscope/components/Card/Card';
import { isModifierKeyPressed } from 'utils/app';

const EXPLORE_EVENT = 'Homepage: Ingestion Active Explore clicked';

interface ActiveIngestionCardProps {
	description: string;
	exploreLabel: string;
	source: string;
	route: string;
}

/**
 * One "ingestion is active" card on the onboarding home. Rendered once per
 * telemetry signal (logs / traces / metrics); the caller supplies the already
 * translated copy plus the analytics source and destination route.
 */
export default function ActiveIngestionCard({
	description,
	exploreLabel,
	source,
	route,
}: ActiveIngestionCardProps): JSX.Element {
	const { safeNavigate } = useSafeNavigate();

	return (
		<Card className="active-ingestion-card" size="small">
			<Card.Content>
				<div className="active-ingestion-card-content-container">
					<div className="active-ingestion-card-content">
						<div className="active-ingestion-card-content-icon">
							<Dot size={16} color={Color.BG_FOREST_500} />
						</div>

						<div className="active-ingestion-card-content-description">
							{description}
						</div>
					</div>

					<div
						role="button"
						tabIndex={0}
						className="active-ingestion-card-actions"
						onClick={(e: React.MouseEvent): void => {
							logEvent(EXPLORE_EVENT, { source });
							safeNavigate(route, { newTab: isModifierKeyPressed(e) });
						}}
						onKeyDown={(e): void => {
							if (e.key === 'Enter') {
								logEvent(EXPLORE_EVENT, { source });
								history.push(route);
							}
						}}
					>
						<Compass size={12} />
						{exploreLabel}
					</div>
				</div>
			</Card.Content>
		</Card>
	);
}
