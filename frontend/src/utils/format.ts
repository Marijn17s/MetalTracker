import {translate} from '../i18n/translations';

const GRAMS_PER_TROY_OUNCE = 31.1034768;
const GRAMS_PER_KILOGRAM = 1000;

let activeIntlLocale: string | undefined;

export function setFormatLocale(locale: string | undefined): void {
  activeIntlLocale = locale;
}

export function formatMoney(value: number, currency: string = 'EUR'): string {
  return new Intl.NumberFormat(activeIntlLocale, {
    style: 'currency',
    currency,
    maximumFractionDigits: 2,
  }).format(value || 0);
}

export function formatPercent(value: number): string {
  const sign = value > 0 ? '+' : '';
  return `${sign}${(value || 0).toFixed(2)}%`;
}

/** Share of a total (e.g. 57% of net) - no +/- prefix. */
export function formatSharePercent(value: number): string {
  return `${(value || 0).toFixed(2)}%`;
}

export function formatWeight(grams: number): string {
  return translate('common.weightGrams', {value: (grams || 0).toFixed(2)});
}

export function formatFineWeight(grams: number): string {
  return translate('common.weightFine', {value: (grams || 0).toFixed(2)});
}

export function metalLabel(metal: string): string {
  if (metal === 'XAU') return translate('common.gold');
  if (metal === 'XAG') return translate('common.silver');
  return metal;
}

export function formLabel(form: string): string {
  if (form === 'coin') return translate('common.coin');
  if (form === 'bar') return translate('common.bar');
  return translate('common.other');
}

export function profitClass(value: number): string {
  if (value > 0) return 'profit-positive';
  if (value < 0) return 'profit-negative';
  return 'profit-neutral';
}

export function formatDate(value?: string): string {
	if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value.slice(0, 10);
  }
  return date.toLocaleDateString(activeIntlLocale);
}

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

export function convertSpotPrice(pricePerKg: number, unit: 'g' | 'troy_oz' | 'kilogram' | string): number {
  const price = pricePerKg || 0;
  if (unit === 'kilogram') {
    return price;
  }
  if (unit === 'g') {
    return price / GRAMS_PER_KILOGRAM;
  }
  return price * (GRAMS_PER_TROY_OUNCE / GRAMS_PER_KILOGRAM);
}

export function spotUnitLabel(unit: 'g' | 'troy_oz' | 'kilogram' | string): string {
  if (unit === 'kilogram') return translate('common.unitKg');
  if (unit === 'g') return translate('common.unitG');
  return translate('common.unitTroyOz');
}

export function costPerSpotUnit(
  costPerKgFine: number,
  unit: 'g' | 'troy_oz' | 'kilogram' | string,
): number {
  return convertSpotPrice(costPerKgFine || 0, unit);
}

export function normalizePurity(purity: number): number {
  if (purity <= 0) return 1;
  if (purity > 1 && purity <= 100) return purity / 100;
  if (purity > 100) return purity / 1000;
  return purity;
}

export function isUnusualPurity(purity: number): boolean {
  const normalized = normalizePurity(purity);
  if (normalized <= 0 || normalized > 1) return true;
  if (normalized >= 0.999) return false;
  const commonValues = [0.995, 0.99, 0.925, 0.9, 0.835, 0.8];
  return !commonValues.some((value) => Math.abs(normalized - value) < 0.001);
}
