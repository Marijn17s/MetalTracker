import {formatDate, formatMoney} from './format';

export function formatChartCurrency(value: number, currency: string): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency,
    maximumFractionDigits: 0,
    minimumFractionDigits: 0,
  }).format(value || 0);
}

/** Compact axis labels (e.g. Oct '25) - full detail stays in tooltips. */
export function formatChartAxisDate(value: string | number): string {
  const text = String(value);
  const parsed = text.length === 7
    ? new Date(`${text}-01T00:00:00`)
    : new Date(text.includes('T') ? text : `${text}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) {
    return text;
  }
  const month = parsed.toLocaleString(undefined, {month: 'short'});
  const year = String(parsed.getFullYear()).slice(-2);
  return `${month} '${year}`;
}

export function formatChartTooltipLabel(value: string | number): string {
  const text = String(value);
  const parsed = text.length === 7
    ? new Date(`${text}-01T00:00:00`)
    : new Date(text.includes('T') ? text : `${text}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) {
    return text;
  }
  if (text.length === 7) {
    return parsed.toLocaleString(undefined, {month: 'long', year: 'numeric'});
  }
  return formatDate(text);
}

export function chartTooltipFormatter(currency: string) {
  return (value: unknown) => {
    const numeric = Array.isArray(value) ? Number(value[0]) : Number(value);
    return formatMoney(Number.isFinite(numeric) ? numeric : 0, currency);
  };
}

export function monthsAgoISO(months: number): string {
  const date = new Date();
  date.setMonth(date.getMonth() - months);
  return date.toISOString().slice(0, 10);
}

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}
