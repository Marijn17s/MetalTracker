import {useEffect, useMemo, useState} from 'react';
import {GetHoldingsFilterOptions, ListGroupedHoldings} from '../../wailsjs/go/main/App';
import {PriceStatusBanner} from '../components/PriceStatusBanner';
import {useLocale} from '../i18n/LocaleContext';
import {Form, GroupedHolding, MetalSymbol} from '../types';
import {formatAppError} from '../utils/errors';
import {
  formatFineWeight,
  formatMoney,
  formatPercent,
  formatWeight,
  formLabel,
  metalLabel,
  profitClass,
} from '../utils/format';

interface HoldingsProps {
  onOpenGroup: (productKey: string) => void;
}

interface FilterOptions {
  brands: string[];
  locations: string[];
}

function toggleValue<T extends string>(values: T[], value: T): T[] {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : [...values, value];
}

export function Holdings({onOpenGroup}: HoldingsProps) {
  const {t} = useLocale();
  const [search, setSearch] = useState('');
  const [metals, setMetals] = useState<MetalSymbol[]>([]);
  const [forms, setForms] = useState<Form[]>([]);
  const [brands, setBrands] = useState<string[]>([]);
  const [locations, setLocations] = useState<string[]>([]);
  const [options, setOptions] = useState<FilterOptions>({brands: [], locations: []});
  const [groups, setGroups] = useState<GroupedHolding[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const activeFilterCount = metals.length + forms.length + brands.length + locations.length;

  useEffect(() => {
    GetHoldingsFilterOptions()
      .then((result) => {
        setOptions({
          brands: result.brands || [],
          locations: result.locations || [],
        });
      })
      .catch((err) => setError(formatAppError(err)));
  }, []);

  useEffect(() => {
    const handle = window.setTimeout(() => {
      setLoading(true);
      ListGroupedHoldings({
        search,
        metals,
        forms,
        brands,
        locations,
      } as never)
        .then((result) => setGroups((result || []) as GroupedHolding[]))
        .catch((err) => setError(formatAppError(err)))
        .finally(() => setLoading(false));
    }, 200);
    return () => window.clearTimeout(handle);
  }, [search, metals, forms, brands, locations]);

  const anyApproximate = groups.some((group) => group.valuationApproximate);

  const emptyMessage = useMemo(() => {
    if (search || activeFilterCount > 0) {
      return {
        title: t('holdings.noMatchTitle'),
        body: t('holdings.noMatchBody'),
      };
    }
    return {
      title: t('holdings.emptyTitle'),
      body: t('holdings.emptyBody'),
    };
  }, [search, activeFilterCount, t]);

  function clearFilters() {
    setSearch('');
    setMetals([]);
    setForms([]);
    setBrands([]);
    setLocations([]);
  }

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('holdings.eyebrow')}</p>
          <h1>{t('holdings.title')}</h1>
        </div>
        <input
          className="search-input"
          placeholder={t('holdings.search')}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
      </header>

      <section className="glass panel holdings-filters">
        <div className="filter-group">
          <span className="filter-label">{t('holdings.metal')}</span>
          <div className="filter-chips">
            {([
              {value: 'XAU' as MetalSymbol, label: t('common.gold')},
              {value: 'XAG' as MetalSymbol, label: t('common.silver')},
            ]).map((item) => (
              <button
                key={item.value}
                type="button"
                className={`range-chip ${metals.includes(item.value) ? 'active' : ''}`}
                onClick={() => setMetals((current) => toggleValue(current, item.value))}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>

        <div className="filter-group">
          <span className="filter-label">{t('holdings.type')}</span>
          <div className="filter-chips">
            {([
              {value: 'bar' as Form, label: t('common.bar')},
              {value: 'coin' as Form, label: t('common.coin')},
              {value: 'other' as Form, label: t('common.other')},
            ]).map((item) => (
              <button
                key={item.value}
                type="button"
                className={`range-chip ${forms.includes(item.value) ? 'active' : ''}`}
                onClick={() => setForms((current) => toggleValue(current, item.value))}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>

        {options.brands.length > 0 && (
          <div className="filter-group">
            <span className="filter-label">{t('holdings.brand')}</span>
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

        {options.locations.length > 0 && (
          <div className="filter-group">
            <span className="filter-label">{t('holdings.location')}</span>
            <div className="filter-chips">
              {options.locations.map((location) => (
                <button
                  key={location}
                  type="button"
                  className={`range-chip ${locations.includes(location) ? 'active' : ''}`}
                  onClick={() => setLocations((current) => toggleValue(current, location))}
                >
                  {location}
                </button>
              ))}
            </div>
          </div>
        )}

        {(search || activeFilterCount > 0) && (
          <div className="filter-actions">
            <button type="button" className="btn btn-ghost" onClick={clearFilters}>
              {t('holdings.clearFilters')}
            </button>
            <span className="muted small">{t('holdings.groupCount', {count: groups.length})}</span>
          </div>
        )}
      </section>

      {error && <p className="error-text">{error}</p>}
      {anyApproximate && (
        <PriceStatusBanner valuationApproximate quoteIsPartial={false} quoteIsStale={false} />
      )}
      {loading && <p className="muted">{t('holdings.loading')}</p>}

      {!loading && groups.length === 0 && (
        <div className="glass panel empty-state">
          <h2>{emptyMessage.title}</h2>
          <p className="muted">{emptyMessage.body}</p>
        </div>
      )}

      <div className="holdings-list">
        {groups.map((group) => {
          const moneyCurrency = group.displayCurrency || group.currency || 'EUR';
          return (
            <button
              key={group.productKey}
              type="button"
              className="glass panel holding-row"
              onClick={() => onOpenGroup(group.productKey)}
            >
              <div>
                <h3>
                  {group.productName || `${metalLabel(group.metal)} ${formLabel(group.form)}`}
                </h3>
                <p className="muted">
                  {metalLabel(group.metal)} - {formLabel(group.form)} - {formatWeight(group.totalWeightGrams || 0)} -{' '}
                  {formatFineWeight(group.totalFineWeightGrams || 0)} -{' '}
                  {group.brand || t('holdings.unknownBrand')} - {t('holdings.heldCount', {count: group.heldCount})}
                </p>
              </div>
            <div className="holding-metrics">
              <strong>{formatMoney(group.totalCurrentWorth, moneyCurrency)}</strong>
              <span className={profitClass(group.totalProfit)}>
                {formatMoney(group.totalProfit, moneyCurrency)} ({formatPercent(group.totalProfitPct)})
              </span>
              <span className="muted small">
                {t('holdings.unrealized', {amount: formatMoney(group.totalUnrealizedProfit || 0, moneyCurrency)})}
              </span>
            </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
