import { ReactNode } from 'react';

export default function NocPanel({
	icon,
	title,
	action,
	onActionClick,
	children,
	className,
}: {
	icon: ReactNode;
	title: string;
	action?: ReactNode;
	onActionClick?: () => void;
	children: ReactNode;
	className?: string;
}): JSX.Element {
	const renderAction = (): ReactNode => {
		if (!action) {
			return null;
		}
		if (onActionClick) {
			return (
				<button
					type="button"
					className="noc-panel-action"
					onClick={onActionClick}
				>
					{action}
				</button>
			);
		}
		return <div className="noc-panel-action">{action}</div>;
	};

	return (
		<div className={`noc-panel${className ? ` ${className}` : ''}`}>
			<div className="noc-panel-head">
				<div className="noc-panel-title">
					<span className="noc-panel-icon">{icon}</span>
					{title}
				</div>
				{renderAction()}
			</div>
			{children}
		</div>
	);
}
