import {useEffect, useMemo, useState} from 'react';
import {GetHoldingsFilterOptions, ListGroupedHoldings} from '../../wailsjs/go/main/App';
import {InventoryToolbar, SortDirection} from '../components/InventoryToolbar';
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

type SortKey = 'profitPct' | 'profit' | 'value' | 'fineWeight' | 'name' | 'heldCount';

function groupDisplayName(group: GroupedHolding): string {
  return group.productName || `${group.metal} ${group.form}`;
}

function compareGroups(
  left: GroupedHolding,
  right: GroupedHolding,
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
      comparison = (left.totalCurrentWorth || 0) - (right.totalCurrentWorth || 0);
      break;
    case 'fineWeight':
      comparison = (left.totalFineWeightGrams || 0) - (right.totalFineWeightGrams || 0);
      break;
    case 'heldCount':
      comparison = (left.heldCount || 0) - (right.heldCount || 0);
      break;
    case 'name':
      comparison = groupDisplayName(left).localeCompare(groupDisplayName(right), undefined, {
        sensitivity: 'base',
      });
      break;
  }
  if (comparison === 0) {
    comparison = left.productKey.localeCompare(right.productKey);
  }
  return sortDirection === 'asc' ? comparison : -comparison;
}

export function Holdings({onOpenGroup}: HoldingsProps) {
  const {t} = useLocale();
  const [search, setSearch] = useState('');
  const [metals, setMetals] = useState<MetalSymbol[]>([]);
  const [forms, setForms] = useState<Form[]>([]);
  const [brands, setBrands] = useState<string[]>([]);
  const [locations, setLocations] = useState<string[]>([]);
  const [sortBy, setSortBy] = useState<SortKey>('profitPct');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
  const [options, setOptions] = useState<FilterOptions>({brands: [], locations: []});
  const [groups, setGroups] = useState<GroupedHolding[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const activeFilterCount = metals.length + forms.length + brands.length + locations.length;
  const sortOptions = [
    {value: 'profitPct' as const, label: t('holdings.sortProfitPct')},
    {value: 'profit' as const, label: t('holdings.sortProfit')},
    {value: 'value' as const, label: t('holdings.sortValue')},
    {value: 'fineWeight' as const, label: t('holdings.sortFineWeight')},
    {value: 'name' as const, label: t('holdings.sortName')},
    {value: 'heldCount' as const, label: t('holdings.sortHeldCount')},
  ];

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

  const sortedGroups = useMemo(
    () => [...groups].sort((left, right) => compareGroups(left, right, sortBy, sortDirection)),
    [groups, sortBy, sortDirection],
  );

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
      </header>

      <InventoryToolbar
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder={t('holdings.search')}
        metals={metals}
        onMetalsChange={setMetals}
        forms={forms}
        onFormsChange={setForms}
        brands={brands}
        onBrandsChange={setBrands}
        brandOptions={options.brands}
        locations={locations}
        onLocationsChange={setLocations}
        locationOptions={options.locations}
        sortBy={sortBy}
        onSortByChange={setSortBy}
        sortDirection={sortDirection}
        onSortDirectionChange={setSortDirection}
        sortOptions={sortOptions}
        resultCountLabel={t('holdings.groupCount', {count: groups.length})}
        onClear={clearFilters}
      />

      {error && <p className="error-text">{error}</p>}
      {anyApproximate && (
        <PriceStatusBanner valuationApproximate quoteIsPartial={false} quoteIsStale={false} />
      )}
      {loading && <p className="muted">{t('holdings.loading')}</p>}

      {!loading && sortedGroups.length === 0 && (
        <div className="glass panel empty-state">
          <h2>{emptyMessage.title}</h2>
          <p className="muted">{emptyMessage.body}</p>
        </div>
      )}

      <div className="holdings-list">
        {sortedGroups.map((group) => {
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
