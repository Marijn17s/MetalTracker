import {useEffect, useMemo, useState} from 'react';
import {GetHoldingsFilterOptions, ListSoldArchive} from '../../wailsjs/go/main/App';
import {InventoryToolbar, SortDirection} from '../components/InventoryToolbar';
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
  weights: number[];
}

type SortKey = 'profitPct' | 'profit' | 'value' | 'fineWeight' | 'name' | 'daysHeld' | 'soldAt';

function unitDisplayName(unit: UnitValuation): string {
  return unit.productName || `${unit.metal} ${unit.form}`;
}

function compareUnits(
  left: UnitValuation,
  right: UnitValuation,
  sortBy: SortKey,
  sortDirection: SortDirection,
): number {
  let comparison = 0;
  switch (sortBy) {
    case 'profitPct':
      comparison = (left.totalProfitPct || 0) - (right.totalProfitPct || 0);
      break;
    case 'profit':
      comparison = (left.totalProfit || 0) - (right.totalProfit || 0);
      break;
    case 'value':
      comparison = (left.salePrice ?? left.currentSpotWorth ?? 0) - (right.salePrice ?? right.currentSpotWorth ?? 0);
      break;
    case 'fineWeight':
      comparison = (left.fineWeightGrams || 0) - (right.fineWeightGrams || 0);
      break;
    case 'daysHeld':
      comparison = (left.daysHeld || 0) - (right.daysHeld || 0);
      break;
    case 'soldAt':
      comparison = (left.soldAt || '').localeCompare(right.soldAt || '');
      break;
    case 'name':
      comparison = unitDisplayName(left).localeCompare(unitDisplayName(right), undefined, {
        sensitivity: 'base',
      });
      break;
  }
  if (comparison === 0) {
    comparison = left.id.localeCompare(right.id);
  }
  return sortDirection === 'asc' ? comparison : -comparison;
}

export function SoldArchive({onOpenUnit}: SoldArchiveProps) {
  const {t} = useLocale();
  const [search, setSearch] = useState('');
  const [metals, setMetals] = useState<MetalSymbol[]>([]);
  const [forms, setForms] = useState<Form[]>([]);
  const [brands, setBrands] = useState<string[]>([]);
  const [weights, setWeights] = useState<number[]>([]);
  const [sortBy, setSortBy] = useState<SortKey>('soldAt');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
  const [options, setOptions] = useState<FilterOptions>({brands: [], weights: []});
  const [units, setUnits] = useState<UnitValuation[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const activeFilterCount = metals.length + forms.length + brands.length + weights.length;
  const sortOptions = [
    {value: 'soldAt' as const, label: t('sold.sortSoldDate')},
    {value: 'profitPct' as const, label: t('holdings.sortProfitPct')},
    {value: 'profit' as const, label: t('holdings.sortProfit')},
    {value: 'value' as const, label: t('sold.sortSalePrice')},
    {value: 'fineWeight' as const, label: t('holdings.sortFineWeight')},
    {value: 'daysHeld' as const, label: t('sold.sortDaysHeld')},
    {value: 'name' as const, label: t('holdings.sortName')},
  ];

  useEffect(() => {
    GetHoldingsFilterOptions({
      search: '',
      metals,
      forms,
      brands: [],
      locations: [],
      weights: [],
    } as never)
      .then((result) => {
        const nextBrands = result.brands || [];
        const nextWeights = result.weights || [];
        setOptions({brands: nextBrands, weights: nextWeights});
        setBrands((current) => current.filter((brand) => nextBrands.includes(brand)));
        setWeights((current) => current.filter((weight) => nextWeights.includes(weight)));
      })
      .catch((err) => setError(formatAppError(err)));
  }, [metals, forms]);

  useEffect(() => {
    const handle = window.setTimeout(() => {
      setLoading(true);
      ListSoldArchive({
        search,
        metals,
        forms,
        brands,
        locations: [],
        weights,
      } as never)
        .then((result) => setUnits((result || []) as UnitValuation[]))
        .catch((err) => setError(formatAppError(err)))
        .finally(() => setLoading(false));
    }, 200);
    return () => window.clearTimeout(handle);
  }, [search, metals, forms, brands, weights]);

  const sortedUnits = useMemo(
    () => [...units].sort((left, right) => compareUnits(left, right, sortBy, sortDirection)),
    [units, sortBy, sortDirection],
  );

  const emptyMessage = useMemo(() => {
    if (search || activeFilterCount > 0) {
      return {
        title: t('sold.noMatchTitle'),
        body: t('sold.noMatchBody'),
      };
    }
    return {
      title: t('sold.emptyTitle'),
      body: t('sold.emptyBody'),
    };
  }, [search, activeFilterCount, t]);

  function clearFilters() {
    setSearch('');
    setMetals([]);
    setForms([]);
    setBrands([]);
    setWeights([]);
  }

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('sold.eyebrow')}</p>
          <h1>{t('sold.title')}</h1>
          <p className="muted">{t('sold.subtitle')}</p>
        </div>
      </header>

      <InventoryToolbar
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder={t('sold.search')}
        metals={metals}
        onMetalsChange={setMetals}
        forms={forms}
        onFormsChange={setForms}
        brands={brands}
        onBrandsChange={setBrands}
        brandOptions={options.brands}
        weights={weights}
        onWeightsChange={setWeights}
        weightOptions={options.weights}
        sortBy={sortBy}
        onSortByChange={setSortBy}
        sortDirection={sortDirection}
        onSortDirectionChange={setSortDirection}
        sortOptions={sortOptions}
        resultCountLabel={t('sold.unitCount', {count: units.length})}
        onClear={clearFilters}
      />

      {error && <p className="error-text">{error}</p>}
      {loading && <p className="muted">{t('sold.loading')}</p>}

      {!loading && sortedUnits.length === 0 && (
        <div className="glass panel empty-state">
          <h2>{emptyMessage.title}</h2>
          <p className="muted">{emptyMessage.body}</p>
        </div>
      )}

      <div className="holdings-list">
        {sortedUnits.map((unit) => {
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
