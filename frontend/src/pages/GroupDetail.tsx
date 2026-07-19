import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
  BulkUpdateHoldingUnits,
  GetSettings,
  ListUnitsInGroup,
  SellUnits,
} from '../../wailsjs/go/main/App';
import {Modal} from '../components/Modal';
import {PriceStatusBanner} from '../components/PriceStatusBanner';
import {useLocale} from '../i18n/LocaleContext';
import {AddInvestmentPrefill, SpotPriceUnit, UnitValuation} from '../types';
import {formatAppError} from '../utils/errors';
import {
  costPerSpotUnit,
  formatDate,
  formatFineWeight,
  formatMoney,
  formatPercent,
  formatWeight,
  formLabel,
  metalLabel,
  profitClass,
  spotUnitLabel,
  todayISO,
} from '../utils/format';

interface GroupDetailProps {
  productKey: string;
  onBack: () => void;
  onOpenUnit: (unitId: string) => void;
  onAddMore: (prefill: AddInvestmentPrefill) => void;
  onChanged: () => void;
}

export function GroupDetail({productKey, onBack, onOpenUnit, onAddMore, onChanged}: GroupDetailProps) {
  const {t} = useLocale();
  const [units, setUnits] = useState<UnitValuation[]>([]);
  const [spotPriceUnit, setSpotPriceUnit] = useState<SpotPriceUnit>('troy_oz');
  const [error, setError] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [showBulkSell, setShowBulkSell] = useState(false);
  const [showBulkEdit, setShowBulkEdit] = useState(false);
  const [busy, setBusy] = useState(false);
  const [soldAt, setSoldAt] = useState(todayISO());
  const [uniformPrice, setUniformPrice] = useState(true);
  const [sharedSalePrice, setSharedSalePrice] = useState(0);
  const [salePrices, setSalePrices] = useState<Record<string, number>>({});
  const [bulkDealer, setBulkDealer] = useState('');
  const [bulkLocation, setBulkLocation] = useState('');
  const [bulkNotes, setBulkNotes] = useState('');

  function load() {
    ListUnitsInGroup(productKey)
      .then((result) => {
        const next = (result || []) as UnitValuation[];
        setUnits(next);
        setSelectedIds((current) => current.filter((id) => next.some((unit) => unit.id === id && unit.status === 'held')));
      })
      .catch((err) => setError(formatAppError(err)));
  }

  useEffect(() => {
    load();
    GetSettings()
      .then((settings) => setSpotPriceUnit((settings.spotPriceUnit as SpotPriceUnit) || 'troy_oz'))
      .catch(() => undefined);
  }, [productKey]);

  const sample = units[0];
  const anyApproximate = units.some((unit) => unit.valuationApproximate);
  const totalWeightGrams = units.reduce((sum, unit) => sum + (unit.weightGrams || 0), 0);
  const totalFineWeight = units.reduce((sum, unit) => sum + (unit.fineWeightGrams || 0), 0);
  const heldUnits = units.filter((unit) => unit.status === 'held');
  const heldPurchase = heldUnits.reduce((sum, unit) => sum + (unit.purchasePrice || 0), 0);
  const heldFine = heldUnits.reduce((sum, unit) => sum + (unit.fineWeightGrams || 0), 0);
  const totalPurchase = units.reduce((sum, unit) => sum + (unit.purchasePrice || 0), 0);
  const breakEvenPerKg = heldFine > 0 ? heldPurchase / (heldFine / 1000) : 0;
  const avgCostPerKg = totalFineWeight > 0 ? totalPurchase / (totalFineWeight / 1000) : 0;
  const currency = sample?.displayCurrency || sample?.currency || 'EUR';
  const unitLabel = spotUnitLabel(spotPriceUnit);

  const selectedHeld = useMemo(
    () => heldUnits.filter((unit) => selectedIds.includes(unit.id)),
    [heldUnits, selectedIds],
  );

  function toggleSelected(unitId: string) {
    setSelectedIds((current) => (
      current.includes(unitId)
        ? current.filter((id) => id !== unitId)
        : [...current, unitId]
    ));
  }

  function selectAllHeld() {
    setSelectedIds(heldUnits.map((unit) => unit.id));
  }

  function clearSelection() {
    setSelectedIds([]);
  }

  function openBulkSell() {
    const defaults: Record<string, number> = {};
    selectedHeld.forEach((unit) => {
      defaults[unit.id] = unit.currentSpotWorth || unit.purchasePrice || 0;
    });
    setSalePrices(defaults);
    setSharedSalePrice(defaults[selectedHeld[0]?.id] || 0);
    setUniformPrice(true);
    setSoldAt(todayISO());
    setShowBulkSell(true);
  }

  function openBulkEdit() {
    const first = selectedHeld[0];
    setBulkDealer(first?.dealer || '');
    setBulkLocation(first?.storageLocation || '');
    setBulkNotes(first?.notes || '');
    setShowBulkEdit(true);
  }

  async function handleBulkSell(event: FormEvent) {
    event.preventDefault();
    if (selectedHeld.length === 0) return;
    setBusy(true);
    setError('');
    try {
      await SellUnits({
        units: selectedHeld.map((unit) => ({
          unitId: unit.id,
          soldAt,
          salePrice: uniformPrice ? sharedSalePrice : (salePrices[unit.id] ?? 0),
        })),
      } as never);
      setShowBulkSell(false);
      clearSelection();
      onChanged();
      load();
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleBulkEdit(event: FormEvent) {
    event.preventDefault();
    if (selectedHeld.length === 0) return;
    setBusy(true);
    setError('');
    try {
      await BulkUpdateHoldingUnits({
        unitIds: selectedHeld.map((unit) => unit.id),
        dealer: bulkDealer,
        storageLocation: bulkLocation,
        notes: bulkNotes,
      } as never);
      setShowBulkEdit(false);
      clearSelection();
      onChanged();
      load();
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  function handleAddMore() {
    if (!sample) return;
    onAddMore({
      metal: sample.metal,
      form: sample.form,
      weightGrams: sample.weightGrams,
      purity: sample.purity <= 1 ? sample.purity * 1000 : sample.purity,
      brand: sample.brand || '',
      productName: sample.productName || '',
      condition: sample.condition || '',
      mintageYear: sample.mintageYear || 0,
    });
  }

  return (
    <div className="page-stack">
      <header className="page-header detail-header">
        <div>
          <button type="button" className="btn btn-ghost back-btn" onClick={onBack}>{t('group.back')}</button>
          <p className="eyebrow">{t('group.eyebrow')}</p>
          <h1>
            {sample
              ? sample.productName || `${metalLabel(sample.metal)} ${formLabel(sample.form)}`
              : t('group.units')}
          </h1>
          {sample && (
            <p className="muted">
              {metalLabel(sample.metal)} - {formLabel(sample.form)} - {formatWeight(totalWeightGrams)} -{' '}
              {formatFineWeight(totalFineWeight)} - {sample.brand || t('common.unknownBrand')}
              {sample.condition ? ` - ${sample.condition}` : ''}
              {sample.mintageYear ? ` - ${sample.mintageYear}` : ''}
            </p>
          )}
        </div>
        {sample && (
          <button type="button" className="btn btn-secondary" onClick={handleAddMore}>
            {t('group.addMore')}
          </button>
        )}
      </header>

      {error && <p className="error-text">{error}</p>}
      {anyApproximate && <PriceStatusBanner valuationApproximate />}

      {sample && (
        <section className="stat-grid">
          <article className="glass panel stat-card">
            <p className="muted">{t('group.breakEven')}</p>
            <strong>
              {heldUnits.length > 0
                ? `${formatMoney(costPerSpotUnit(breakEvenPerKg, spotPriceUnit), currency)} / ${unitLabel}`
                : t('common.emDash')}
            </strong>
          </article>
          <article className="glass panel stat-card">
            <p className="muted">{t('group.avgCost', {unit: unitLabel})}</p>
            <strong>
              {formatMoney(costPerSpotUnit(avgCostPerKg, spotPriceUnit), currency)} / {unitLabel}
            </strong>
          </article>
        </section>
      )}

      {heldUnits.length > 0 && (
        <div className="bulk-toolbar glass panel">
          <div className="bulk-toolbar-left">
            <button type="button" className="btn btn-ghost" onClick={selectAllHeld}>
              {t('group.selectAll')}
            </button>
            {selectedIds.length > 0 && (
              <button type="button" className="btn btn-ghost" onClick={clearSelection}>
                {t('group.clear')}
              </button>
            )}
            <span className="muted small">
              {t('group.selected', {count: selectedHeld.length})}
            </span>
          </div>
          <div className="bulk-toolbar-right">
            <button
              type="button"
              className="btn btn-secondary"
              disabled={selectedHeld.length === 0}
              onClick={openBulkEdit}
            >
              {t('group.bulkEdit')}
            </button>
            <button
              type="button"
              className="btn btn-primary"
              disabled={selectedHeld.length === 0}
              onClick={openBulkSell}
            >
              {t('group.bulkSell')}
            </button>
          </div>
        </div>
      )}

      <div className="holdings-list">
        {units.map((unit) => {
          const moneyCurrency = unit.displayCurrency || unit.currency;
          const isHeld = unit.status === 'held';
          const isSelected = selectedIds.includes(unit.id);
          return (
            <div
              key={unit.id}
              className={`glass panel holding-row holding-row-selectable ${isSelected ? 'selected' : ''}`}
            >
              {isHeld && (
                <label className="unit-select" onClick={(event) => event.stopPropagation()}>
                  <input
                    type="checkbox"
                    checked={isSelected}
                    onChange={() => toggleSelected(unit.id)}
                    aria-label={t('group.selectAria', {date: formatDate(unit.purchasedAt)})}
                  />
                </label>
              )}
              <button
                type="button"
                className="holding-row-main"
                onClick={() => onOpenUnit(unit.id)}
              >
                <div>
                  <h3>{unit.status === 'sold' ? t('group.soldUnit') : t('group.heldUnit')}</h3>
                  <p className="muted">
                    {t('group.bought')} {formatDate(unit.purchasedAt)}
                    {unit.soldAt ? ` - ${t('group.sold')} ${formatDate(unit.soldAt)}` : ''}
                    {' - '}
                    {formatFineWeight(unit.fineWeightGrams || 0)}
                    {unit.daysHeld ? ` - ${t('sold.daysHeld', {days: unit.daysHeld})}` : ''}
                    {unit.daysHeld ? ` - ${t('sold.annualized', {pct: formatPercent(unit.annualizedReturnPct || 0)})}` : ''}
                  </p>
                </div>
                <div className="holding-metrics">
                  <strong>{formatMoney(unit.currentSpotWorth, moneyCurrency)}</strong>
                  <span className={profitClass(unit.totalProfit)}>
                    {formatMoney(unit.totalProfit, moneyCurrency)} ({formatPercent(unit.totalProfitPct)})
                    {' '}{unit.isRealized ? t('group.realized') : t('group.unrealized')}
                  </span>
                </div>
              </button>
            </div>
          );
        })}
      </div>

      {showBulkSell && (
        <Modal onBackdropClick={() => !busy && setShowBulkSell(false)}>
          <form className="stack" onSubmit={handleBulkSell}>
            <h2>{t('group.bulkSellTitle')}</h2>
            <p className="muted small">{t('group.bulkSellBody')}</p>
            <label>
              {t('unit.saleDate')}
              <input type="date" value={soldAt} onChange={(event) => setSoldAt(event.target.value)} required />
            </label>
            <label className="checkbox-row">
              <input
                type="checkbox"
                checked={uniformPrice}
                onChange={(event) => setUniformPrice(event.target.checked)}
              />
              {t('group.sameSalePrice')}
            </label>
            {uniformPrice ? (
              <label>
                {t('unit.salePrice')} ({selectedHeld[0]?.currency || currency})
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={sharedSalePrice}
                  onChange={(event) => setSharedSalePrice(Number(event.target.value))}
                  required
                />
              </label>
            ) : (
              <div className="stack">
                {selectedHeld.map((unit) => (
                  <label key={unit.id}>
                    {t('sold.bought', {date: formatDate(unit.purchasedAt)})} ({unit.currency})
                    <input
                      type="number"
                      min="0"
                      step="any"
                      value={salePrices[unit.id] ?? 0}
                      onChange={(event) => setSalePrices((current) => ({
                        ...current,
                        [unit.id]: Number(event.target.value),
                      }))}
                      required
                    />
                  </label>
                ))}
              </div>
            )}
            <div className="button-row">
              <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => setShowBulkSell(false)}>
                {t('common.cancel')}
              </button>
              <button type="submit" className="btn btn-primary" disabled={busy}>
                {busy ? t('group.selling') : t('group.confirmSales')}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {showBulkEdit && (
        <Modal onBackdropClick={() => !busy && setShowBulkEdit(false)}>
          <form className="stack" onSubmit={handleBulkEdit}>
            <h2>{t('group.bulkEditTitle')}</h2>
            <p className="muted small">{t('group.bulkEditBody')}</p>
            <label>
              {t('group.dealer')}
              <input value={bulkDealer} onChange={(event) => setBulkDealer(event.target.value)} />
            </label>
            <label>
              {t('group.storage')}
              <input value={bulkLocation} onChange={(event) => setBulkLocation(event.target.value)} />
            </label>
            <label>
              {t('group.notes')}
              <input value={bulkNotes} onChange={(event) => setBulkNotes(event.target.value)} />
            </label>
            <div className="button-row">
              <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => setShowBulkEdit(false)}>
                {t('common.cancel')}
              </button>
              <button type="submit" className="btn btn-primary" disabled={busy}>
                {busy ? t('add.saving') : t('group.apply')}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
}
