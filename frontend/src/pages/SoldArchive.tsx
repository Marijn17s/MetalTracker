import {useEffect, useMemo, useState} from 'react';
import {GetHoldingsFilterOptions, ListSoldArchive} from '../../wailsjs/go/main/App';
import {useLocale} from '../i18n/LocaleContext';
import {Form, MetalSymbol, UnitValuation} from '../types';
import {formatAppError} from '../utils/errors';
import {
  formatDate,
  formatFineWeight,
  formatMoney,
  formatPercent,
  formLabel,
  metalLabel,
  profitClass,
} from '../utils/format';

interface SoldArchiveProps {
  onOpenUnit: (unitId: string, productKey: string) => void;
}

interface FilterOptions {
  brands: string[];
}

function toggleValue<T extends string>(values: T[], value: T): T[] {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : [...values, value];
}

export function SoldArchive({onOpenUnit}: SoldArchiveProps) {
  const {t} = useLocale();
  const [search, setSearch] = useState('');
  const [metals, setMetals] = useState<MetalSymbol[]>([]);
  const [forms, setForms] = useState<Form[]>([]);
  const [brands, setBrands] = useState<string[]>([]);
  const [options, setOptions] = useState<FilterOptions>({brands: []});
  const [units, setUnits] = useState<UnitValuation[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    GetHoldingsFilterOptions()
      .then((result) => setOptions({brands: result.brands || []}))
      .catch((err) => setError(formatAppError(err)));
  }, []);

  useEffect(() => {
    const handle = window.setTimeout(() => {
      setLoading(true);
      ListSoldArchive({
        search,
        metals,
        forms,
        brands,
        locations: [],
      } as never)
        .then((result) => setUnits((result || []) as UnitValuation[]))
        .catch((err) => setError(formatAppError(err)))
        .finally(() => setLoading(false));
    }, 200);
    return () => window.clearTimeout(handle);
  }, [search, metals, forms, brands]);

  const emptyMessage = useMemo(() => {
    if (search || metals.length || forms.length || brands.length) {
      return {
        title: t('sold.noMatchTitle'),
        body: t('sold.noMatchBody'),
      };
    }
    return {
      title: t('sold.emptyTitle'),
      body: t('sold.emptyBody'),
    };
  }, [search, metals, forms, brands, t]);

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('sold.eyebrow')}</p>
          <h1>{t('sold.title')}</h1>
          <p className="muted">{t('sold.subtitle')}</p>
        </div>
      </header>

      <section className="glass panel holdings-filters">
        <input
          className="search-input"
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t('sold.search')}
          aria-label={t('sold.search')}
        />
        <div className="filter-group">
          <span className="filter-label">{t('sold.metal')}</span>
          <div className="filter-chips">
            {(['XAU', 'XAG'] as MetalSymbol[]).map((metal) => (
              <button
                key={metal}
                type="button"
                className={`range-chip ${metals.includes(metal) ? 'active' : ''}`}
                onClick={() => setMetals((current) => toggleValue(current, metal))}
              >
                {metalLabel(metal)}
              </button>
            ))}
          </div>
        </div>
        <div className="filter-group">
          <span className="filter-label">{t('sold.type')}</span>
          <div className="filter-chips">
            {(['bar', 'coin', 'other'] as Form[]).map((form) => (
              <button
                key={form}
                type="button"
                className={`range-chip ${forms.includes(form) ? 'active' : ''}`}
                onClick={() => setForms((current) => toggleValue(current, form))}
              >
                {formLabel(form)}
              </button>
            ))}
          </div>
        </div>
        {options.brands.length > 0 && (
          <div className="filter-group">
            <span className="filter-label">{t('sold.brand')}</span>
            <div className="filter-chips">
              {options.brands.map((brand) => (
                <button
                  key={brand}
                  type="button"
                  className={`range-chip ${brands.includes(brand) ? 'active' : ''}`}
                  onClick={() => setBrands((current) => toggleValue(current, brand))}
                >
                  {brand}
                </button>
              ))}
            </div>
          </div>
        )}
      </section>

      {error && <p className="error-text">{error}</p>}
      {loading && <p className="muted">{t('sold.loading')}</p>}

      {!loading && units.length === 0 && (
        <div className="glass panel empty-state">
          <h2>{emptyMessage.title}</h2>
          <p className="muted">{emptyMessage.body}</p>
        </div>
      )}

      <div className="holdings-list">
        {units.map((unit) => {
          const moneyCurrency = unit.displayCurrency || unit.currency;
          return (
            <button
              key={unit.id}
              type="button"
              className="glass panel holding-row"
              onClick={() => onOpenUnit(unit.id, unit.productKey)}
            >
              <div>
                <h3>
                  {unit.productName || `${metalLabel(unit.metal)} ${formLabel(unit.form)}`}
                </h3>
                <p className="muted">
                  {metalLabel(unit.metal)} - {formLabel(unit.form)} - {unit.brand || t('common.unknownBrand')}
                  {' - '}{t('sold.bought', {date: formatDate(unit.purchasedAt)})}
                  {unit.soldAt ? ` - ${t('sold.soldOn', {date: formatDate(unit.soldAt)})}` : ''}
                  {' - '}{formatFineWeight(unit.fineWeightGrams || 0)}
                  {unit.daysHeld ? ` - ${t('sold.daysHeld', {days: unit.daysHeld})}` : ''}
                </p>
              </div>
              <div className="holding-metrics">
                <strong>{formatMoney(unit.salePrice ?? unit.currentSpotWorth, moneyCurrency)}</strong>
                <span className={profitClass(unit.totalProfit)}>
                  {formatMoney(unit.totalProfit, moneyCurrency)} ({formatPercent(unit.totalProfitPct)}) {t('sold.realized')}
                </span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
