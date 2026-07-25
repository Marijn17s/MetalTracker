import {useState} from 'react';
import {useLocale} from '../i18n/LocaleContext';
import {Form, MetalSymbol} from '../types';

export type SortDirection = 'asc' | 'desc';

export interface SortOption<T extends string> {
  value: T;
  label: string;
}

interface InventoryToolbarProps<TSort extends string> {
  search: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder: string;
  metals: MetalSymbol[];
  onMetalsChange: (value: MetalSymbol[]) => void;
  forms: Form[];
  onFormsChange: (value: Form[]) => void;
  brands: string[];
  onBrandsChange: (value: string[]) => void;
  brandOptions: string[];
  locations?: string[];
  onLocationsChange?: (value: string[]) => void;
  locationOptions?: string[];
  sortBy: TSort;
  onSortByChange: (value: TSort) => void;
  sortDirection: SortDirection;
  onSortDirectionChange: (value: SortDirection) => void;
  sortOptions: SortOption<TSort>[];
  resultCountLabel: string;
  onClear: () => void;
}

function toggleValue<T extends string>(values: T[], value: T): T[] {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : [...values, value];
}

export function InventoryToolbar<TSort extends string>({
  search,
  onSearchChange,
  searchPlaceholder,
  metals,
  onMetalsChange,
  forms,
  onFormsChange,
  brands,
  onBrandsChange,
  brandOptions,
  locations,
  onLocationsChange,
  locationOptions,
  sortBy,
  onSortByChange,
  sortDirection,
  onSortDirectionChange,
  sortOptions,
  resultCountLabel,
  onClear,
}: InventoryToolbarProps<TSort>) {
  const {t} = useLocale();
  const [filtersOpen, setFiltersOpen] = useState(false);

  const locationValues = locations || [];
  const availableLocations = locationOptions || [];
  const showLocations = Boolean(onLocationsChange && availableLocations.length > 0);
  const activeFilterCount =
    metals.length + forms.length + brands.length + (showLocations ? locationValues.length : 0);
  const hasActiveQuery = Boolean(search || activeFilterCount > 0);

  function handleClear() {
    onClear();
    setFiltersOpen(false);
  }

  return (
    <section className="glass panel holdings-toolbar">
      <div className="holdings-toolbar-main">
        <input
          className="search-input holdings-toolbar-search"
          placeholder={searchPlaceholder}
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          aria-label={searchPlaceholder}
        />
        <div className="holdings-sort-controls">
          <label className="holdings-sort-field">
            <select
              value={sortBy}
              onChange={(event) => onSortByChange(event.target.value as TSort)}
              aria-label={t('holdings.sort')}
            >
              {sortOptions.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            className="range-chip holdings-sort-direction"
            onClick={() => onSortDirectionChange(sortDirection === 'asc' ? 'desc' : 'asc')}
            aria-label={sortDirection === 'asc' ? t('holdings.sortAsc') : t('holdings.sortDesc')}
            title={sortDirection === 'asc' ? t('holdings.sortAsc') : t('holdings.sortDesc')}
          >
            {sortDirection === 'asc' ? '↑' : '↓'}
          </button>
        </div>
        <button
          type="button"
          className={`range-chip ${filtersOpen || activeFilterCount > 0 ? 'active' : ''}`}
          onClick={() => setFiltersOpen((current) => !current)}
          aria-expanded={filtersOpen}
        >
          {activeFilterCount > 0
            ? t('holdings.filtersActive', {count: activeFilterCount})
            : t('holdings.filters')}
        </button>
      </div>

      {filtersOpen && (
        <div className="holdings-filters-panel">
          <div className="filter-group filter-group-inline">
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
                  onClick={() => onMetalsChange(toggleValue(metals, item.value))}
                >
                  {item.label}
                </button>
              ))}
            </div>
          </div>

          <div className="filter-group filter-group-inline">
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
                  onClick={() => onFormsChange(toggleValue(forms, item.value))}
                >
                  {item.label}
                </button>
              ))}
            </div>
          </div>

          {brandOptions.length > 0 && (
            <div className="filter-group filter-group-inline">
              <span className="filter-label">{t('holdings.brand')}</span>
              <div className="filter-chips">
                {brandOptions.map((brand) => (
                  <button
                    key={brand}
                    type="button"
                    className={`range-chip ${brands.includes(brand) ? 'active' : ''}`}
                    onClick={() => onBrandsChange(toggleValue(brands, brand))}
                  >
                    {brand}
                  </button>
                ))}
              </div>
            </div>
          )}

          {showLocations && onLocationsChange && (
            <div className="filter-group filter-group-inline">
              <span className="filter-label">{t('holdings.location')}</span>
              <div className="filter-chips">
                {availableLocations.map((location) => (
                  <button
                    key={location}
                    type="button"
                    className={`range-chip ${locationValues.includes(location) ? 'active' : ''}`}
                    onClick={() => onLocationsChange(toggleValue(locationValues, location))}
                  >
                    {location}
                  </button>
                ))}
              </div>
            </div>
          )}

          {hasActiveQuery && (
            <div className="filter-actions">
              <button type="button" className="btn btn-ghost" onClick={handleClear}>
                {t('holdings.clearFilters')}
              </button>
              <span className="muted small">{resultCountLabel}</span>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
