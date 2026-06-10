import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Color } from '@signozhq/design-tokens';
import { Switch } from 'antd';
import logEvent from 'api/common/logEvent';
import { Delete } from 'lucide-react';
import { useTimezone } from 'providers/Timezone';

import './TimezoneAdaptation.styles.scss';

function TimezoneAdaptation(): JSX.Element {
	const { t } = useTranslation(['settings']);
	const {
		timezone,
		browserTimezone,
		updateTimezone,
		isAdaptationEnabled,
		setIsAdaptationEnabled,
	} = useTimezone();

	const isTimezoneOverridden = useMemo(
		() => timezone.offset !== browserTimezone.offset,
		[timezone, browserTimezone],
	);

	const getSwitchStyles = (): React.CSSProperties => ({
		backgroundColor:
			isAdaptationEnabled && isTimezoneOverridden ? Color.BG_AMBER_400 : undefined,
	});

	const handleOverrideClear = (): void => {
		updateTimezone(browserTimezone);
		logEvent('Account Settings: Timezone override cleared', {});
	};

	const handleSwitchChange = (): void => {
		setIsAdaptationEnabled((prev) => {
			const isEnabled = !prev;
			logEvent(
				`Account Settings: Timezone adaptation ${
					isEnabled ? 'enabled' : 'disabled'
				}`,
				{},
			);
			return isEnabled;
		});
	};

	return (
		<div className="timezone-adaption">
			<div className="timezone-adaption__header">
				<h2 className="timezone-adaption__title">
					{t('settings:timezone_adapt_title')}
				</h2>
				<Switch
					checked={isAdaptationEnabled}
					onChange={handleSwitchChange}
					style={getSwitchStyles()}
					data-testid="timezone-adaptation-switch"
				/>
			</div>

			<p className="timezone-adaption__description">
				{t('settings:timezone_adapt_description')}
			</p>

			<div className="timezone-adaption__note">
				<div className="timezone-adaption__note-text-container">
					<span className="timezone-adaption__bullet">•</span>
					<span className="timezone-adaption__note-text">
						{isTimezoneOverridden ? (
							<>
								{t('settings:timezone_overridden_to')}
								<span className="timezone-adaption__note-text-overridden">
									{timezone.offset}
								</span>
							</>
						) : (
							<>{t('settings:timezone_override_hint')}</>
						)}
					</span>
				</div>

				{!!isTimezoneOverridden && (
					<button
						type="button"
						className="timezone-adaption__clear-override"
						onClick={handleOverrideClear}
					>
						<Delete height={12} width={12} color={Color.BG_ROBIN_300} />
						{t('settings:timezone_clear_override')}
					</button>
				)}
			</div>
		</div>
	);
}

export default TimezoneAdaptation;
