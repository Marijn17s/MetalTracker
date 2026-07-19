import {useLocale} from '../i18n/LocaleContext';
import {monthsAgoISO, todayISO} from '../utils/chart';

export type RangePreset = '3m' | '6m' | '12m' | 'custom';

interface DateRangeControlsProps {
  preset: RangePreset;
  fromDate: string;
  toDate: string;
  onPresetChange: (preset: RangePreset) => void;
  onFromChange: (value: string) => void;
  onToChange: (value: string) => void;
}

export function resolvePresetRange(preset: RangePreset): {fromDate: string; toDate: string} {
  if (preset === '3m') return {fromDate: monthsAgoISO(3), toDate: todayISO()};
  if (preset === '6m') return {fromDate: monthsAgoISO(6), toDate: todayISO()};
  if (preset === '12m') return {fromDate: monthsAgoISO(12), toDate: todayISO()};
  return {fromDate: monthsAgoISO(12), toDate: todayISO()};
}

export function DateRangeControls({
  preset,
  fromDate,
  toDate,
  onPresetChange,
  onFromChange,
  onToChange,
}: DateRangeControlsProps) {
  const {t} = useLocale();

  return (
    <div className="range-controls">
      <div className="range-presets">
        {(['3m', '6m', '12m', 'custom'] as RangePreset[]).map((item) => (
          <button
            key={item}
            type="button"
            className={`range-chip ${preset === item ? 'active' : ''}`}
            onClick={() => onPresetChange(item)}
          >
            {item === 'custom' ? t('range.custom') : item.toUpperCase()}
          </button>
        ))}
      </div>
      {preset === 'custom' && (
        <div className="range-dates">
          <label>
            {t('range.from')}
            <input type="date" value={fromDate} onChange={(event) => onFromChange(event.target.value)} />
          </label>
          <label>
            {t('range.to')}
            <input type="date" value={toDate} onChange={(event) => onToChange(event.target.value)} />
          </label>
        </div>
      )}
    </div>
  );
}
