import React from 'react';
import { Button } from '@signozhq/ui';
import logEvent from 'api/common/logEvent';
import { useSafeNavigate } from 'hooks/useSafeNavigate';
import Card from 'periscope/components/Card/Card';
import { isModifierKeyPressed } from 'utils/app';

const EXPLORE_EVENT = 'Homepage: Explore clicked';

export interface ExplorerAction {
	label: string;
	/** Button.prefix가 ReactElement만 받는다 — ReactNode로 두면 tsc TS2322. */
	icon: React.ReactElement;
	source: string;
	route: string;
}

interface ExplorerActionCardProps {
	iconUrl: string;
	iconAlt: string;
	title: string;
	description: string;
	actions: ExplorerAction[];
	/** Original markup lazy-loads the explorer and alert icons but not the dashboard one. */
	lazyIcon?: boolean;
}

/**
 * A titled section card on the onboarding home carrying one or more navigation
 * buttons (explorers, dashboard creation, alert creation).
 */
export default function ExplorerActionCard({
	iconUrl,
	iconAlt,
	title,
	description,
	actions,
	lazyIcon,
}: ExplorerActionCardProps): JSX.Element {
	const { safeNavigate } = useSafeNavigate();

	return (
		<Card className="explorer-card">
			<Card.Content>
				<div className="section-container">
					<div className="section-content">
						<div className="section-icon">
							<img
								src={iconUrl}
								alt={iconAlt}
								width={16}
								height={16}
								loading={lazyIcon ? 'lazy' : undefined}
							/>
						</div>

						<div className="section-title">
							<div className="title">{title}</div>

							<div className="description">{description}</div>
						</div>
					</div>

					<div className="section-actions">
						{actions.map((action) => (
							<Button
								key={`${action.source}-${action.route}`}
								variant="solid"
								color="secondary"
								className="periscope-btn secondary"
								prefix={action.icon}
								onClick={(e: React.MouseEvent): void => {
									logEvent(EXPLORE_EVENT, { source: action.source });
									safeNavigate(action.route, { newTab: isModifierKeyPressed(e) });
								}}
							>
								{action.label}
							</Button>
						))}
					</div>
				</div>
			</Card.Content>
		</Card>
	);
}
