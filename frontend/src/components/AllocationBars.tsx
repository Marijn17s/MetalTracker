import {useLocale} from '../i18n/LocaleContext';
import {AllocationSlice, Currency} from '../types';
import {formatMoney} from '../utils/format';

interface AllocationBarsProps {
  title: string;
  slices: AllocationSlice[];
  currency: Currency | string;
  emptyLabel?: string;
}

export function AllocationBars({
  title,
  slices,
  currency,
  emptyLabel,
}: AllocationBarsProps) {
  const {t} = useLocale();
  const resolvedEmptyLabel = emptyLabel ?? t('dashboard.noHeldUnits');

  return (
    <article className="glass panel">
      <h2>{title}</h2>
      {slices.length === 0 && <p className="muted">{resolvedEmptyLabel}</p>}
      <div className="allocation-list">
        {slices.map((slice) => (
          <div key={slice.key} className="allocation-row">
            <div className="allocation-meta">
              <span>{slice.label}</span>
              <strong>
                {formatMoney(slice.worth, currency)} - {slice.percent.toFixed(1)}%
              </strong>
            </div>
            <div className="allocation-bar" aria-hidden="true">
              <div className="allocation-bar-fill" style={{width: `${Math.min(100, Math.max(0, slice.percent))}%`}} />
            </div>
          </div>
        ))}
      </div>
    </article>
  );
}
