import {FormEvent, useEffect, useState} from 'react';
import {PreviewWhatIf} from '../../wailsjs/go/main/App';
import {useLocale} from '../i18n/LocaleContext';
import {SpotPriceUnit, WhatIfPreview} from '../types';
import {formatAppError} from '../utils/errors';
import {
  convertSpotPrice,
  formatMoney,
  formatPercent,
  profitClass,
  spotUnitLabel,
} from '../utils/format';

interface WhatIfPanelProps {
  spotPriceUnit: SpotPriceUnit;
  baselineGoldPerKg: number;
  baselineSilverPerKg: number;
  currency: string;
}

export function WhatIfPanel({
  spotPriceUnit,
  baselineGoldPerKg,
  baselineSilverPerKg,
  currency,
}: WhatIfPanelProps) {
  const {t} = useLocale();
  const unitLabel = spotUnitLabel(spotPriceUnit);
  const [goldSpot, setGoldSpot] = useState(0);
  const [silverSpot, setSilverSpot] = useState(0);
  const [preview, setPreview] = useState<WhatIfPreview | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    setGoldSpot(Number(convertSpotPrice(baselineGoldPerKg, spotPriceUnit).toFixed(2)));
    setSilverSpot(Number(convertSpotPrice(baselineSilverPerKg, spotPriceUnit).toFixed(2)));
    setPreview(null);
  }, [baselineGoldPerKg, baselineSilverPerKg, spotPriceUnit]);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError('');
    try {
      const result = await PreviewWhatIf({
        goldSpot,
        silverSpot,
        spotUnit: spotPriceUnit,
      } as never);
      setPreview(result as WhatIfPreview);
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <article className="glass panel">
      <h2>{t('whatif.title')}</h2>
      <p className="muted small">{t('whatif.body')}</p>
      <form className="form-grid what-if-form" onSubmit={handleSubmit}>
        <label>
          {t('whatif.gold', {unit: unitLabel})}
          <input
            type="number"
            min="0"
            step="any"
            value={goldSpot || ''}
            onChange={(event) => setGoldSpot(Number(event.target.value) || 0)}
          />
        </label>
        <label>
          {t('whatif.silver', {unit: unitLabel})}
          <input
            type="number"
            min="0"
            step="any"
            value={silverSpot || ''}
            onChange={(event) => setSilverSpot(Number(event.target.value) || 0)}
          />
        </label>
        <div className="span-2 form-actions">
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? t('whatif.calculating') : t('whatif.preview')}
          </button>
        </div>
      </form>
      {error && <p className="error-text">{error}</p>}
      {preview && (
        <div className="split-stats what-if-results">
          <div>
            <p className="muted">{t('whatif.worth')}</p>
            <strong>{formatMoney(preview.portfolioWorth, currency)}</strong>
          </div>
          <div>
            <p className="muted">{t('whatif.unrealized')}</p>
            <strong className={profitClass(preview.unrealizedProfit)}>
              {formatMoney(preview.unrealizedProfit, currency)} ({formatPercent(preview.unrealizedProfitPct)})
            </strong>
          </div>
          <div>
            <p className="muted">{t('whatif.vsCurrent')}</p>
            <strong className={profitClass(preview.worthDelta)}>
              {formatMoney(preview.worthDelta, currency)}
            </strong>
          </div>
        </div>
      )}
    </article>
  );
}
