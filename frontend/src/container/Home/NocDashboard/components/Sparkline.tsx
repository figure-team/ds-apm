export default function Sparkline({
	points,
	color,
	width = 46,
	height = 18,
}: {
	points: number[];
	color: string;
	width?: number;
	height?: number;
}): JSX.Element {
	const pad = 2;
	const w = width - pad * 2;
	const h = height - pad * 2;
	const step = points.length > 1 ? w / (points.length - 1) : 0;
	const coords = points
		.map((p, i) => {
			const x = pad + i * step;
			const y = pad + (1 - Math.max(0, Math.min(1, p))) * h;
			return `${x.toFixed(1)},${y.toFixed(1)}`;
		})
		.join(' ');

	return (
		<svg
			className="noc-spark"
			width={width}
			height={height}
			viewBox={`0 0 ${width} ${height}`}
			aria-hidden
		>
			<polyline points={coords} fill="none" stroke={color} strokeWidth={1.5} />
		</svg>
	);
}
