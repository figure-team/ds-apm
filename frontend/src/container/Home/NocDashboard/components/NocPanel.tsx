import { ReactNode } from 'react';

export default function NocPanel({
	icon,
	title,
	action,
	children,
	className,
}: {
	icon: ReactNode;
	title: string;
	action?: ReactNode;
	children: ReactNode;
	className?: string;
}): JSX.Element {
	return (
		<div className={`noc-panel${className ? ` ${className}` : ''}`}>
			<div className="noc-panel-head">
				<div className="noc-panel-title">
					<span className="noc-panel-icon">{icon}</span>
					{title}
				</div>
				{action ? <div className="noc-panel-action">{action}</div> : null}
			</div>
			{children}
		</div>
	);
}
