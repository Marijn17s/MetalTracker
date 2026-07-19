import {useEffect, useMemo, useState} from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import {GetMonthlyBreakdown, GetMonthlyPage, GetSettings} from '../../wailsjs/go/main/App';
import {DateRangeControls, RangePreset, resolvePresetRange} from '../components/DateRangeControls';
import {MonthlyMetalBreakdown, PnLContributionReport} from '../types';
import {
  chartTooltipFormatter,
  formatChartAxisDate,
  formatChartCurrency,
  formatChartTooltipLabel,
  monthsAgoISO,
} from '../utils/chart';
import {useLocale} from '../i18n/LocaleContext';
import {formatAppError} from '../utils/errors';
import {formatMoney, formatPercent, formatSharePercent, metalLabel, profitClass} from '../utils/format';

type ChartMode = 'absolute' | 'percent';

function shiftMonths(isoDate: string, deltaMonths: number): string {
  const date = new Date(`${isoDate}T00:00:00Z`);
  date.setUTCMonth(date.getUTCMonth() + deltaMonths);
  return date.toISOString().slice(0, 10);
}

function mergeMonthlyRows(
  current: MonthlyMetalBreakdown[],
  incoming: MonthlyMetalBreakdown[],
): MonthlyMetalBreakdown[] {
  const map = new Map<string, MonthlyMetalBreakdown>();
  for (const row of [...current, ...incoming]) {
    map.set(`${row.yearMonth}|${row.metal}`, row);
  }
  return Array.from(map.values()).sort((left, right) => {
    if (left.yearMonth === right.yearMonth) {
      return left.metal.localeCompare(right.metal);
    }
    return right.yearMonth.localeCompare(left.yearMonth);
  });
}

function applyPreset(
  next: RangePreset,
  setPreset: (preset: RangePreset) => void,
  setFromDate: (value: string) => void,
  setToDate: (value: string) => void,
) {
  setPreset(next);
  if (next !== 'custom') {
    const range = resolvePresetRange(next);
    setFromDate(range.fromDate);
    setToDate(range.toDate);
  }
}

export function Monthly() {
  const {t} = useLocale();
  const [currency, setCurrency] = useState('EUR');
  const [error, setError] = useState('');

  const initialRange = resolvePresetRange('12m');

  // Chart + P&L contribution share this range.
  const [chartPreset, setChartPreset] = useState<RangePreset>('12m');
  const [chartFromDate, setChartFromDate] = useState(initialRange.fromDate);
  const [chartToDate, setChartToDate] = useState(initialRange.toDate);
  const [chartMode, setChartMode] = useState<ChartMode>('absolute');
  const [chartRows, setChartRows] = useState<MonthlyMetalBreakdown[]>([]);
  const [contribution, setContribution] = useState<PnLContributionReport | null>(null);
  const [loadingOverview, setLoadingOverview] = useState(false);

  // Month-by-month list is independent.
  const [listRows, setListRows] = useState<MonthlyMetalBreakdown[]>([]);
  const [listFromDate, setListFromDate] = useState(initialRange.fromDate);
  const [loadingList, setLoadingList] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  useEffect(() => {
    GetSettings()
      .then((settings) => setCurrency(settings.displayCurrency))
      .catch((err) => setError(formatAppError(err)));
  }, []);

  useEffect(() => {
    const range = chartPreset === 'custom'
      ? {fromDate: chartFromDate, toDate: chartToDate}
      : resolvePresetRange(chartPreset);
    setLoadingOverview(true);
    setError('');
    GetMonthlyPage(range.fromDate, range.toDate)
      .then((page) => {
        setChartRows((page?.breakdown || []) as MonthlyMetalBreakdown[]);
        setContribution((page?.contribution || null) as PnLContributionReport | null);
      })
      .catch((err) => setError(formatAppError(err)))
      .finally(() => setLoadingOverview(false));
  }, [chartPreset, chartFromDate, chartToDate]);

  useEffect(() => {
    const range = resolvePresetRange('12m');
    setLoadingList(true);
    GetMonthlyBreakdown(range.fromDate, range.toDate)
      .then((breakdown) => {
        setListRows((breakdown || []) as MonthlyMetalBreakdown[]);
        setListFromDate(range.fromDate);
      })
      .catch((err) => setError(formatAppError(err)))
      .finally(() => setLoadingList(false));
  }, []);

  async function handleLoadMore() {
    const olderFrom = shiftMonths(listFromDate, -6);
    const olderTo = shiftMonths(listFromDate, -1);
    setLoadingMore(true);
    setError('');
    try {
      const breakdown = await GetMonthlyBreakdown(olderFrom, olderTo);
      setListRows((current) => mergeMonthlyRows(current, (breakdown || []) as MonthlyMetalBreakdown[]));
      setListFromDate(olderFrom);
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setLoadingMore(false);
    }
  }

  const chartData = useMemo(() => {
    const byMonth = new Map<string, {
      yearMonth: string;
      gold: number;
      silver: number;
    }>();
    for (const row of [...chartRows].reverse()) {
      const existing = byMonth.get(row.yearMonth) || {yearMonth: row.yearMonth, gold: 0, silver: 0};
      const value = chartMode === 'percent' ? row.netPct : row.netChange;
      if (row.metal === 'XAU') {
        existing.gold = value;
      } else {
        existing.silver = value;
      }
      byMonth.set(row.yearMonth, existing);
    }
    return Array.from(byMonth.values());
  }, [chartRows, chartMode]);

  const axisColor = getComputedStyle(document.documentElement).getPropertyValue('--muted').trim() || '#b8b2a8';
  const gridColor = getComputedStyle(document.documentElement).getPropertyValue('--chart-grid').trim() || 'rgba(255,255,255,0.08)';
  const tooltipBg = getComputedStyle(document.documentElement).getPropertyValue('--tooltip-bg').trim() || 'rgba(24,22,20,0.92)';
  const tooltipBorder = getComputedStyle(document.documentElement).getPropertyValue('--glass-border').trim() || 'rgba(255,255,255,0.1)';

  const overviewEmpty = !loadingOverview && chartRows.length === 0;
  const listEmpty = !loadingList && listRows.length === 0;

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('monthly.eyebrow')}</p>
          <h1>{t('monthly.title')}</h1>
          <p className="muted">{t('monthly.subtitle')}</p>
        </div>
      </header>

      {error && <p className="error-text">{error}</p>}

      {overviewEmpty && listEmpty && (
        <div className="glass panel empty-state">
          <h2>{t('monthly.emptyTitle')}</h2>
          <p className="muted">{t('monthly.emptyBody')}</p>
        </div>
      )}

      <article className="glass panel chart-panel">
        <div className="panel-header-row">
          <h2>{t('monthly.chartTitle')}</h2>
          <div className="chart-controls">
            <DateRangeControls
              preset={chartPreset}
              fromDate={chartFromDate}
              toDate={chartToDate}
              onPresetChange={(next) => applyPreset(next, setChartPreset, setChartFromDate, setChartToDate)}
              onFromChange={setChartFromDate}
              onToChange={setChartToDate}
            />
            <div className="range-presets">
              <button
                type="button"
                className={`range-chip ${chartMode === 'absolute' ? 'active' : ''}`}
                onClick={() => setChartMode('absolute')}
              >
                {currency}
              </button>
              <button
                type="button"
                className={`range-chip ${chartMode === 'percent' ? 'active' : ''}`}
                onClick={() => setChartMode('percent')}
              >
                %
              </button>
            </div>
          </div>
        </div>
        <div className="chart-wrap">
          <ResponsiveContainer width="100%" height={280}>
            <BarChart data={chartData} margin={{left: 4, right: 12, top: 8, bottom: 28}}>
              <CartesianGrid stroke={gridColor} />
              <XAxis
                dataKey="yearMonth"
                stroke={axisColor}
                tick={{fontSize: 11}}
                tickFormatter={formatChartAxisDate}
                interval={0}
                angle={-40}
                textAnchor="end"
                height={56}
                tickMargin={8}
                padding={{left: 8, right: 8}}
              />
              <YAxis
                stroke={axisColor}
                tick={{fontSize: 12}}
                tickFormatter={(value) => (
                  chartMode === 'percent'
                    ? `${Number(value).toFixed(1)}%`
                    : formatChartCurrency(Number(value), currency)
                )}
                width={80}
              />
              <Tooltip
                formatter={((value: unknown) => (
                  chartMode === 'percent'
                    ? formatPercent(Number(value) || 0)
                    : chartTooltipFormatter(currency)(value)
                )) as never}
                labelFormatter={(label) => formatChartTooltipLabel(String(label))}
                contentStyle={{
                  background: tooltipBg,
                  border: `1px solid ${tooltipBorder}`,
                  borderRadius: 12,
                }}
              />
              <Legend />
              <Bar dataKey="gold" name={t('monthly.gold')} fill="#d4af37" radius={[6, 6, 0, 0]} />
              <Bar dataKey="silver" name={t('monthly.silver')} fill="#9aa0a8" radius={[6, 6, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </article>

      {contribution && (
        <section className="content-grid">
          <article className="glass panel">
            <h2>{t('monthly.contributionTitle')}</h2>
            <p className="muted small">
              {t('monthly.contributionBody')}
            </p>
            <div className="spot-list">
              {(contribution.byMetal || []).map((item) => (
                <div key={item.key}>
                  <span>{item.label}</span>
                  <strong className={profitClass(item.netChange)}>
                    {formatMoney(item.netChange, currency)} ({formatSharePercent(item.percentOfTotalNet)})
                  </strong>
                </div>
              ))}
              {(contribution.byMetal || []).length === 0 && <p className="muted">{t('monthly.contributionEmpty')}</p>}
            </div>
            <div className="split-stats">
              <div>
                <p className="muted">{t('monthly.unrealized')}</p>
                <strong className={profitClass(contribution.totalUnrealized)}>
                  {formatMoney(contribution.totalUnrealized, currency)}
                </strong>
              </div>
              <div>
                <p className="muted">{t('monthly.realized')}</p>
                <strong className={profitClass(contribution.totalRealized)}>
                  {formatMoney(contribution.totalRealized, currency)}
                </strong>
              </div>
            </div>
          </article>
          <article className="glass panel">
            <h2>{t('monthly.topGroupsTitle')}</h2>
            <p className="muted small">{t('monthly.topGroupsBody')}</p>
            <div className="spot-list">
              {(contribution.byGroup || []).map((item) => (
                <div key={item.key}>
                  <span>{item.label}</span>
                  <strong className={profitClass(item.netChange)}>
                    {formatMoney(item.netChange, currency)}
                    <span className="muted small"> - {t('monthly.detailUnrealized', {amount: formatMoney(item.unrealizedChange, currency)})} - {t('monthly.detailRealized', {amount: formatMoney(item.realizedProfit, currency)})}</span>
                  </strong>
                </div>
              ))}
              {(contribution.byGroup || []).length === 0 && <p className="muted">{t('monthly.topGroupsEmpty')}</p>}
            </div>
          </article>
        </section>
      )}

      <div className="table-wrap glass panel">
        <div className="panel-header-row">
          <h2>{t('monthly.listTitle')}</h2>
        </div>
        <table>
          <thead>
            <tr>
              <th>{t('monthly.colMonth')}</th>
              <th>{t('monthly.colMetal')}</th>
              <th>{t('monthly.colUnrealized')}</th>
              <th>{t('monthly.colNet')}</th>
              <th>{t('monthly.colRealized')}</th>
            </tr>
          </thead>
          <tbody>
            {listRows.map((row) => (
              <tr key={`${row.yearMonth}-${row.metal}`}>
                <td>{row.yearMonth}</td>
                <td>{metalLabel(row.metal)}</td>
                <td className={profitClass(row.unrealizedChange)}>
                  <div>{formatMoney(row.unrealizedChange, currency)}</div>
                  <div className="muted small">{formatPercent(row.unrealizedPct)}</div>
                </td>
                <td className={profitClass(row.netChange)}>
                  <div>{formatMoney(row.netChange, currency)}</div>
                  <div className="muted small">{formatPercent(row.netPct)}</div>
                </td>
                <td className={profitClass(row.realizedProfit)}>
                  <div>{formatMoney(row.realizedProfit, currency)}</div>
                  <div className="muted small">{formatPercent(row.realizedPct)}</div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="load-more-row">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={handleLoadMore}
            disabled={loadingList || loadingMore}
          >
            {loadingMore ? t('monthly.loadingMore') : t('monthly.loadMore')}
          </button>
          <span className="muted small">
            {t('monthly.listFrom', {date: listFromDate || monthsAgoISO(12)})}
          </span>
        </div>
      </div>
    </div>
  );
}
