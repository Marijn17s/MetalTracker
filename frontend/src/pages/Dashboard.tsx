import {useEffect, useState} from 'react';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import {
  GetAllocationBreakdown,
  GetMetalAverageCosts,
  GetPortfolioHistory,
  GetPortfolioSummary,
  GetPortfolioValueAt,
  GetSettings,
} from '../../wailsjs/go/main/App';
import {AllocationBars} from '../components/AllocationBars';
import {DateRangeControls, RangePreset, resolvePresetRange} from '../components/DateRangeControls';
import {PriceStatusBanner} from '../components/PriceStatusBanner';
import {WhatIfPanel} from '../components/WhatIfPanel';
import {useLocale} from '../i18n/LocaleContext';
import {
  AllocationBreakdown,
  AppSettings,
  AppView,
  MetalAverageCost,
  PortfolioHistoryPoint,
  PortfolioSummary,
  PortfolioValueAt,
  SpotPriceUnit,
} from '../types';
import {
  chartTooltipFormatter,
  formatChartAxisDate,
  formatChartCurrency,
  formatChartTooltipLabel,
  todayISO,
} from '../utils/chart';
import {formatAppError} from '../utils/errors';
import {
  convertSpotPrice,
  costPerSpotUnit,
  formatMoney,
  formatPercent,
  formatWeight,
  metalLabel,
  profitClass,
  spotUnitLabel,
} from '../utils/format';

interface DashboardProps {
  onNavigate: (view: AppView) => void;
}

type ChartSeriesMode = 'total' | 'gold' | 'silver' | 'both';

export function Dashboard({onNavigate}: DashboardProps) {
  const {t} = useLocale();
  const [summary, setSummary] = useState<PortfolioSummary | null>(null);
  const [history, setHistory] = useState<PortfolioHistoryPoint[]>([]);
  const [allocation, setAllocation] = useState<AllocationBreakdown | null>(null);
  const [averageCosts, setAverageCosts] = useState<MetalAverageCost[]>([]);
  const [spotPriceUnit, setSpotPriceUnit] = useState<SpotPriceUnit>('troy_oz');
  const [error, setError] = useState('');
  const [preset, setPreset] = useState<RangePreset>('12m');
  const initialRange = resolvePresetRange('12m');
  const [fromDate, setFromDate] = useState(initialRange.fromDate);
  const [toDate, setToDate] = useState(initialRange.toDate);
  const [chartSeries, setChartSeries] = useState<ChartSeriesMode>('total');
  const [valueAtDate, setValueAtDate] = useState(todayISO());
  const [valueAt, setValueAt] = useState<PortfolioValueAt | null>(null);
  const [valueAtLoading, setValueAtLoading] = useState(false);
  const [valueAtError, setValueAtError] = useState('');

  useEffect(() => {
    GetPortfolioSummary()
      .then((portfolioSummary) => setSummary(portfolioSummary as PortfolioSummary))
      .catch((err) => setError(formatAppError(err)));
    GetAllocationBreakdown()
      .then((result) => setAllocation(result as AllocationBreakdown))
      .catch(() => undefined);
    GetMetalAverageCosts()
      .then((result) => setAverageCosts((result || []) as MetalAverageCost[]))
      .catch(() => undefined);
    GetSettings()
      .then((settings) => {
        const loaded = settings as AppSettings;
        setSpotPriceUnit(loaded.spotPriceUnit || 'troy_oz');
      })
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    const range = preset === 'custom' ? {fromDate, toDate} : resolvePresetRange(preset);
    GetPortfolioHistory(range.fromDate, range.toDate)
      .then((portfolioHistory) => setHistory((portfolioHistory || []) as PortfolioHistoryPoint[]))
      .catch((err) => setError(formatAppError(err)));
  }, [preset, fromDate, toDate]);

  useEffect(() => {
    if (!valueAtDate) {
      return;
    }
    let cancelled = false;
    setValueAtLoading(true);
    setValueAtError('');
    GetPortfolioValueAt(valueAtDate)
      .then((result) => {
        if (!cancelled) {
          setValueAt(result as PortfolioValueAt);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setValueAt(null);
          setValueAtError(formatAppError(err));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setValueAtLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [valueAtDate]);

  if (error && !summary) {
    return <div className="glass panel"><p className="error-text">{error}</p></div>;
  }

  if (!summary) {
    return <div className="glass panel"><p className="muted">{t('dashboard.loading')}</p></div>;
  }

  const currency = summary.displayCurrency;
  const unitLabel = spotUnitLabel(spotPriceUnit);
  const goldSpot = convertSpotPrice(summary.goldSpotPerKg, spotPriceUnit);
  const silverSpot = convertSpotPrice(summary.silverSpotPerKg, spotPriceUnit);
  const axisColor = getComputedStyle(document.documentElement).getPropertyValue('--muted').trim() || '#b8b2a8';
  const gridColor = getComputedStyle(document.documentElement).getPropertyValue('--chart-grid').trim() || 'rgba(255,255,255,0.08)';
  const tooltipBg = getComputedStyle(document.documentElement).getPropertyValue('--tooltip-bg').trim() || 'rgba(24,22,20,0.92)';
  const tooltipBorder = getComputedStyle(document.documentElement).getPropertyValue('--glass-border').trim() || 'rgba(255,255,255,0.1)';
  const isEmpty = summary.heldUnits === 0 && summary.soldUnits === 0;
  const valueAtCurrency = valueAt?.displayCurrency || currency;

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('dashboard.eyebrow')}</p>
          <h1>{t('dashboard.title')}</h1>
        </div>
      </header>

      {isEmpty && (
        <section className="glass panel">
          <h2>{t('dashboard.onboardingTitle')}</h2>
          <p className="muted">{t('dashboard.onboardingBody')}</p>
          <ol className="onboarding-list">
            <li>{t('dashboard.onboardingApi')}</li>
            <li>{t('dashboard.onboardingAdd')}</li>
            <li>{t('dashboard.onboardingHoldings')}</li>
          </ol>
          <div className="button-row">
            <button type="button" className="btn btn-primary" onClick={() => onNavigate('add')}>
              {t('dashboard.ctaAdd')}
            </button>
            <button type="button" className="btn btn-ghost" onClick={() => onNavigate('settings')}>
              {t('dashboard.ctaSettings')}
            </button>
          </div>
        </section>
      )}

      <PriceStatusBanner
        quoteAsOf={summary.quoteAsOf}
        quoteIsStale={summary.quoteIsStale}
        quoteIsPartial={summary.quoteIsPartial}
        valuationApproximate={summary.valuationApproximate}
        priceErrorCode={summary.priceErrorCode}
      />

      <section className="stat-grid">
        <article className="glass panel stat-card">
          <p className="muted">{t('dashboard.currentWorth')}</p>
          <strong>{formatMoney(summary.totalCurrentWorth, currency)}</strong>
        </article>
        <article className="glass panel stat-card">
          <p className="muted">{t('dashboard.totalCost')}</p>
          <strong>{formatMoney(summary.totalPurchaseCost, currency)}</strong>
        </article>
        <article className="glass panel stat-card">
          <p className="muted">{t('dashboard.totalPnl')}</p>
          <strong className={profitClass(summary.totalProfit)}>
            {formatMoney(summary.totalProfit, currency)} ({formatPercent(summary.totalProfitPct)})
          </strong>
        </article>
        <article className="glass panel stat-card">
          <p className="muted">{t('dashboard.units')}</p>
          <strong>
            {t('dashboard.heldSold', {held: summary.heldUnits, sold: summary.soldUnits})}
          </strong>
          {(summary.heldGoldFineWeightGrams > 0 || summary.heldSilverFineWeightGrams > 0) && (
            <p className="muted small">
              {[
                summary.heldGoldFineWeightGrams > 0
                  ? t('dashboard.heldMetalGrams', {
                      metal: t('common.gold'),
                      weight: formatWeight(summary.heldGoldFineWeightGrams),
                    })
                  : null,
                summary.heldSilverFineWeightGrams > 0
                  ? t('dashboard.heldMetalGrams', {
                      metal: t('common.silver'),
                      weight: formatWeight(summary.heldSilverFineWeightGrams),
                    })
                  : null,
              ]
                .filter(Boolean)
                .join(' - ')}
            </p>
          )}
        </article>
      </section>

      <section className="content-grid">
        <article className="glass panel chart-panel">
          <div className="panel-header-row">
            <h2>{t('dashboard.valueOverTime')}</h2>
            <div className="chart-controls">
              <div className="range-presets">
                {([
                  {id: 'total', label: t('dashboard.chartSeriesTotal')},
                  {id: 'gold', label: t('dashboard.chartSeriesGold')},
                  {id: 'silver', label: t('dashboard.chartSeriesSilver')},
                  {id: 'both', label: t('dashboard.chartSeriesBoth')},
                ] as {id: ChartSeriesMode; label: string}[]).map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className={`range-chip ${chartSeries === item.id ? 'active' : ''}`}
                    onClick={() => setChartSeries(item.id)}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
              <DateRangeControls
                preset={preset}
                fromDate={fromDate}
                toDate={toDate}
                onPresetChange={(next) => {
                  setPreset(next);
                  if (next !== 'custom') {
                    const range = resolvePresetRange(next);
                    setFromDate(range.fromDate);
                    setToDate(range.toDate);
                  }
                }}
                onFromChange={setFromDate}
                onToChange={setToDate}
              />
            </div>
          </div>
          <div className="chart-wrap">
            <ResponsiveContainer width="100%" height={280}>
              <LineChart data={history} margin={{left: 4, right: 12, top: 8, bottom: 28}}>
                <CartesianGrid stroke={gridColor} />
                <XAxis
                  dataKey="date"
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
                  tickFormatter={(value) => formatChartCurrency(Number(value), currency)}
                  width={80}
                />
                <Tooltip
                  formatter={chartTooltipFormatter(currency) as never}
                  labelFormatter={(label) => formatChartTooltipLabel(String(label))}
                  contentStyle={{
                    background: tooltipBg,
                    border: `1px solid ${tooltipBorder}`,
                    borderRadius: 12,
                  }}
                />
                <Legend />
                {chartSeries === 'total' && (
                  <>
                    <Line type="monotone" dataKey="portfolioWorth" name={t('dashboard.worth')} stroke="#d4af37" strokeWidth={2.5} dot={false} />
                    <Line type="monotone" dataKey="costBasis" name={t('dashboard.cost')} stroke="#8a8f98" strokeWidth={2} dot={false} />
                  </>
                )}
                {chartSeries === 'gold' && (
                  <>
                    <Line type="monotone" dataKey="goldWorth" name={t('dashboard.chartGoldWorth')} stroke="#d4af37" strokeWidth={2.5} dot={false} />
                    <Line type="monotone" dataKey="goldCostBasis" name={t('dashboard.chartGoldCost')} stroke="#8a8f98" strokeWidth={2} dot={false} />
                  </>
                )}
                {chartSeries === 'silver' && (
                  <>
                    <Line type="monotone" dataKey="silverWorth" name={t('dashboard.chartSilverWorth')} stroke="#c0c5ce" strokeWidth={2.5} dot={false} />
                    <Line type="monotone" dataKey="silverCostBasis" name={t('dashboard.chartSilverCost')} stroke="#8a8f98" strokeWidth={2} dot={false} />
                  </>
                )}
                {chartSeries === 'both' && (
                  <>
                    <Line type="monotone" dataKey="goldWorth" name={t('dashboard.chartGoldWorth')} stroke="#d4af37" strokeWidth={2.5} dot={false} />
                    <Line type="monotone" dataKey="silverWorth" name={t('dashboard.chartSilverWorth')} stroke="#c0c5ce" strokeWidth={2.5} dot={false} />
                  </>
                )}
              </LineChart>
            </ResponsiveContainer>
          </div>
        </article>

        <article className="glass panel">
          <h2>{t('dashboard.spotPrices')}</h2>
          <div className="spot-list">
            <div>
              <span>{metalLabel('XAU')}</span>
              <strong>{formatMoney(goldSpot, currency)} / {unitLabel}</strong>
            </div>
            <div>
              <span>{metalLabel('XAG')}</span>
              <strong>{formatMoney(silverSpot, currency)} / {unitLabel}</strong>
            </div>
          </div>
          <div className="split-stats">
            <div>
              <p className="muted">{t('dashboard.unrealized')}</p>
              <strong className={profitClass(summary.totalUnrealizedProfit)}>
                {formatMoney(summary.totalUnrealizedProfit, currency)}
              </strong>
            </div>
            <div>
              <p className="muted">{t('dashboard.realized')}</p>
              <strong className={profitClass(summary.totalRealizedProfit)}>
                {formatMoney(summary.totalRealizedProfit, currency)}
              </strong>
            </div>
          </div>
        </article>
      </section>

      {!isEmpty && (
        <section className="glass panel">
          <div className="panel-header-row">
            <div>
              <h2>{t('dashboard.worthOnDate')}</h2>
              <p className="muted small">{t('dashboard.worthOnDateBody')}</p>
            </div>
            <div className="range-dates">
              <label>
                {t('dashboard.worthOnDatePick')}
                <input
                  type="date"
                  value={valueAtDate}
                  max={todayISO()}
                  onChange={(event) => setValueAtDate(event.target.value)}
                />
              </label>
            </div>
          </div>
          {valueAtLoading && <p className="muted">{t('dashboard.worthOnDateLoading')}</p>}
          {valueAtError && <p className="error-text">{valueAtError}</p>}
          {!valueAtLoading && valueAt && (
            <>
              <PriceStatusBanner
                quoteAsOf={valueAt.quoteAsOf}
                quoteIsStale={valueAt.quoteIsStale}
                quoteIsPartial={valueAt.quoteIsPartial}
                valuationApproximate={valueAt.valuationApproximate}
                priceErrorCode={valueAt.priceErrorCode}
              />
              <div className="stat-grid">
                <article className="stat-card">
                  <p className="muted">{t('dashboard.worth')}</p>
                  <strong>{formatMoney(valueAt.portfolioWorth, valueAtCurrency)}</strong>
                </article>
                <article className="stat-card">
                  <p className="muted">{t('dashboard.cost')}</p>
                  <strong>{formatMoney(valueAt.costBasis, valueAtCurrency)}</strong>
                </article>
                <article className="stat-card">
                  <p className="muted">{t('dashboard.unrealized')}</p>
                  <strong className={profitClass(valueAt.unrealizedProfit)}>
                    {formatMoney(valueAt.unrealizedProfit, valueAtCurrency)} ({formatPercent(valueAt.unrealizedProfitPct)})
                  </strong>
                </article>
                <article className="stat-card">
                  <p className="muted">{t('dashboard.units')}</p>
                  <strong>{t('dashboard.worthOnDateHeld', {count: valueAt.heldUnits})}</strong>
                </article>
              </div>
              {(valueAt.goldWorth > 0 || valueAt.silverWorth > 0) && (
                <div className="spot-list">
                  {valueAt.goldWorth > 0 && (
                    <div>
                      <span>{t('dashboard.worthOnDateGold')}</span>
                      <strong>{formatMoney(valueAt.goldWorth, valueAtCurrency)}</strong>
                    </div>
                  )}
                  {valueAt.silverWorth > 0 && (
                    <div>
                      <span>{t('dashboard.worthOnDateSilver')}</span>
                      <strong>{formatMoney(valueAt.silverWorth, valueAtCurrency)}</strong>
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </section>
      )}

      {!isEmpty && (
        <>
          <section className="analytics-grid">
            <AllocationBars
              title={t('dashboard.allocationMetal')}
              slices={allocation?.byMetal || []}
              currency={currency}
            />
            <AllocationBars
              title={t('dashboard.allocationForm')}
              slices={allocation?.byForm || []}
              currency={currency}
            />
            <AllocationBars
              title={t('dashboard.allocationBrand')}
              slices={allocation?.byBrand || []}
              currency={currency}
            />
            <AllocationBars
              title={t('dashboard.allocationLocation')}
              slices={allocation?.byLocation || []}
              currency={currency}
            />
          </section>

          <section className="content-grid">
            <article className="glass panel">
              <h2>{t('dashboard.avgCostTitle', {unit: unitLabel})}</h2>
              <p className="muted small">{t('dashboard.avgCostBody')}</p>
              {averageCosts.length === 0 && <p className="muted">{t('dashboard.noHeldUnits')}</p>}
              <div className="spot-list">
                {averageCosts.map((item) => (
                  <div key={item.metal}>
                    <span>{metalLabel(item.metal)}</span>
                    <strong>
                      {formatMoney(costPerSpotUnit(item.avgCostPerKgFine, spotPriceUnit), currency)} / {unitLabel}
                    </strong>
                  </div>
                ))}
              </div>
            </article>
            <WhatIfPanel
              spotPriceUnit={spotPriceUnit}
              baselineGoldPerKg={summary.goldSpotPerKg}
              baselineSilverPerKg={summary.silverSpotPerKg}
              currency={currency}
            />
          </section>
        </>
      )}

      {error && <p className="error-text">{error}</p>}
    </div>
  );
}
