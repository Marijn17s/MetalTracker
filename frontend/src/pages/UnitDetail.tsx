import {FormEvent, useEffect, useState} from 'react';
import {
  DeleteHoldingUnit,
  GetSettings,
  GetUnit,
  SellUnit,
  UpdateHoldingUnit,
} from '../../wailsjs/go/main/App';
import {AttachmentsPanel} from '../components/AttachmentsPanel';
import {ConfirmDialog} from '../components/ConfirmDialog';
import {Modal} from '../components/Modal';
import {PriceStatusBanner} from '../components/PriceStatusBanner';
import {useLocale} from '../i18n/LocaleContext';
import {
  AddInvestmentPrefill,
  Currency,
  Form,
  MetalSymbol,
  SpotPriceUnit,
  UnitStatus,
  UnitValuation,
  WeightUnit,
} from '../types';
import {formatAppError} from '../utils/errors';
import {
  costPerSpotUnit,
  formatDate,
  formatFineWeight,
  formatMoney,
  formatPercent,
  formatWeight,
  formLabel,
  isUnusualPurity,
  metalLabel,
  profitClass,
  spotUnitLabel,
  todayISO,
} from '../utils/format';

interface UnitDetailProps {
  unitId: string;
  onBack: () => void;
  onChanged: () => void;
  onDeleted: (unitId: string) => void;
  onAddMore: (prefill: AddInvestmentPrefill) => void;
}

interface EditFormState {
  metal: MetalSymbol;
  form: Form;
  weight: number;
  weightUnit: WeightUnit;
  purity: number;
  brand: string;
  productName: string;
  purchasePrice: number;
  spotWorthAtPurchase: number;
  isGift: boolean;
  purchasedAt: string;
  notes: string;
  dealer: string;
  status: UnitStatus;
  soldAt: string;
  salePrice: number;
  storageLocation: string;
  condition: string;
  mintageYear: number;
}

function toDateInput(value?: string): string {
  if (!value) return todayISO();
  return value.slice(0, 10);
}

function toEditForm(unit: UnitValuation): EditFormState {
  return {
    metal: unit.metal,
    form: unit.form,
    weight: unit.weightGrams,
    weightUnit: 'g',
    purity: unit.purity <= 1 ? unit.purity * 1000 : unit.purity,
    brand: unit.brand,
    productName: unit.productName,
    purchasePrice: unit.purchasePrice,
    spotWorthAtPurchase: unit.spotWorthAtPurchase,
    isGift: Boolean(unit.isGift),
    purchasedAt: toDateInput(unit.purchasedAt),
    notes: unit.notes,
    dealer: unit.dealer,
    status: unit.status,
    soldAt: toDateInput(unit.soldAt) || todayISO(),
    salePrice: unit.salePrice ?? unit.purchasePrice,
    storageLocation: unit.storageLocation || '',
    condition: unit.condition || '',
    mintageYear: unit.mintageYear || 0,
  };
}

export function UnitDetail({unitId, onBack, onChanged, onDeleted, onAddMore}: UnitDetailProps) {
  const {t} = useLocale();
  const [unit, setUnit] = useState<UnitValuation | null>(null);
  const [error, setError] = useState('');
  const [showSell, setShowSell] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [soldAt, setSoldAt] = useState(todayISO());
  const [salePrice, setSalePrice] = useState(0);
  const [editForm, setEditForm] = useState<EditFormState | null>(null);
  const [busy, setBusy] = useState(false);
  const [spotPriceUnit, setSpotPriceUnit] = useState<SpotPriceUnit>('troy_oz');

  function load() {
    GetUnit(unitId)
      .then((result) => {
        const valuation = result as UnitValuation;
        setUnit(valuation);
        setSalePrice(valuation.currentSpotWorth || valuation.purchasePrice);
        setEditForm(toEditForm(valuation));
      })
      .catch((err) => setError(formatAppError(err)));
  }

  useEffect(() => {
    load();
    GetSettings()
      .then((settings) => setSpotPriceUnit((settings.spotPriceUnit as SpotPriceUnit) || 'troy_oz'))
      .catch(() => undefined);
  }, [unitId]);

  async function handleSell(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError('');
    try {
      await SellUnit({
        unitId,
        soldAt,
        salePrice,
      } as never);
      setShowSell(false);
      onChanged();
      load();
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleEdit(event: FormEvent) {
    event.preventDefault();
    if (!editForm) return;
    setBusy(true);
    setError('');
    try {
      await UpdateHoldingUnit({
        unitId,
        metal: editForm.metal,
        form: editForm.form,
        weight: editForm.weight,
        weightUnit: editForm.weightUnit,
        purity: editForm.purity,
        brand: editForm.brand,
        productName: editForm.productName,
        purchasePrice: editForm.isGift ? 0 : editForm.purchasePrice,
        spotWorthAtPurchase: editForm.spotWorthAtPurchase,
        isGift: editForm.isGift,
        purchasedAt: editForm.purchasedAt,
        notes: editForm.notes,
        dealer: editForm.dealer,
        status: editForm.status,
        soldAt: editForm.status === 'sold' ? editForm.soldAt : '',
        salePrice: editForm.status === 'sold' ? editForm.salePrice : undefined,
        storageLocation: editForm.storageLocation,
        condition: editForm.condition,
        mintageYear: editForm.mintageYear,
      } as never);
      setShowEdit(false);
      onChanged();
      load();
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete() {
    setBusy(true);
    setError('');
    try {
      await DeleteHoldingUnit(unitId);
      setShowDelete(false);
      onDeleted(unitId);
    } catch (err) {
      setError(formatAppError(err));
      setBusy(false);
    }
  }

  if (!unit || !editForm) {
    return <div className="glass panel"><p className="muted">{t('unit.loading')}</p></div>;
  }

  const currency = (unit.displayCurrency || unit.currency) as Currency;
  const purchaseCurrency = unit.currency as Currency;
  const unitLabel = spotUnitLabel(spotPriceUnit);
  const breakEven = unit.breakEvenSpotPerKg
    ? costPerSpotUnit(unit.breakEvenSpotPerKg, spotPriceUnit)
    : 0;

  return (
    <div className="page-stack">
      <header className="page-header detail-header">
        <div>
          <button type="button" className="btn btn-ghost back-btn" onClick={onBack}>{t('unit.back')}</button>
          <p className="eyebrow">{t('unit.eyebrow')}</p>
          <h1>{unit.productName || `${metalLabel(unit.metal)} ${formLabel(unit.form)}`}</h1>
          <p className="muted">
            {metalLabel(unit.metal)} - {formLabel(unit.form)} - {formatWeight(unit.weightGrams)} -{' '}
            {formatFineWeight(unit.fineWeightGrams || 0)} - {unit.brand || t('common.unknownBrand')}
            {unit.condition ? ` - ${unit.condition}` : ''}
            {unit.mintageYear ? ` - ${unit.mintageYear}` : ''}
            {unit.storageLocation ? ` - ${unit.storageLocation}` : ''}
          </p>
        </div>
        <div className="header-actions">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => onAddMore({
              metal: unit.metal,
              form: unit.form,
              weightGrams: unit.weightGrams,
              purity: unit.purity <= 1 ? unit.purity * 1000 : unit.purity,
              brand: unit.brand || '',
              productName: unit.productName || '',
              condition: unit.condition || '',
              mintageYear: unit.mintageYear || 0,
            })}
          >
            {t('unit.addMore')}
          </button>
          <button type="button" className="btn btn-ghost" onClick={() => {
            setEditForm(toEditForm(unit));
            setShowEdit(true);
          }}>
            {t('unit.edit')}
          </button>
          <button type="button" className="btn btn-ghost btn-danger-text" onClick={() => setShowDelete(true)}>
            {t('unit.delete')}
          </button>
          {unit.status === 'held' && (
            <button type="button" className="btn btn-primary" onClick={() => setShowSell(true)}>
              {t('unit.markSold')}
            </button>
          )}
        </div>
      </header>

      {unit.valuationApproximate && <PriceStatusBanner valuationApproximate />}

      <section className="stat-grid">
        <article className="glass panel stat-card">
          <p className="muted">{t('unit.purchasePrice')} ({currency})</p>
          <strong>{unit.isGift ? t('unit.gift') : formatMoney(unit.purchasePrice, currency)}</strong>
        </article>
        <article className="glass panel stat-card">
          <p className="muted">{t('unit.spotAtPurchase')}</p>
          <strong>{formatMoney(unit.spotWorthAtPurchase, currency)}</strong>
        </article>
        <article className="glass panel stat-card">
          <p className="muted">{unit.isRealized ? t('unit.salePrice') : t('unit.currentWorth')}</p>
          <strong>{formatMoney(unit.currentSpotWorth, currency)}</strong>
        </article>
        <article className="glass panel stat-card">
          <p className="muted">{unit.isRealized ? t('unit.realizedPnl') : t('unit.unrealizedPnl')}</p>
          <strong className={profitClass(unit.totalProfit)}>
            {formatMoney(unit.totalProfit, currency)} ({formatPercent(unit.totalProfitPct)})
          </strong>
        </article>
      </section>

      <section className="glass panel">
        <h2>{t('unit.details')}</h2>
        <dl className="detail-list">
          <div><dt>{t('unit.purchased')}</dt><dd>{formatDate(unit.purchasedAt)}</dd></div>
          <div>
            <dt>{t('unit.status')}</dt>
            <dd>
              {unit.status === 'sold' ? t('unit.statusSold') : t('unit.statusHeld')}
              {unit.isGift ? ` · ${t('unit.gift')}` : ''}
            </dd>
          </div>
          <div><dt>{t('unit.fineWeight')}</dt><dd>{formatFineWeight(unit.fineWeightGrams || 0)}</dd></div>
          {unit.soldAt && <div><dt>{t('unit.sold')}</dt><dd>{formatDate(unit.soldAt)}</dd></div>}
          {unit.salePrice != null && (
            <div><dt>{t('unit.salePrice')}</dt><dd>{formatMoney(unit.salePrice, currency)}</dd></div>
          )}
          <div><dt>{t('unit.premiumPaid')}</dt><dd>{formatMoney(unit.premiumPaid, currency)}</dd></div>
          <div><dt>{t('unit.metalDelta')}</dt><dd className={profitClass(unit.metalDelta)}>{formatMoney(unit.metalDelta, currency)}</dd></div>
          <div><dt>{t('unit.purchaseCurrency')}</dt><dd>{purchaseCurrency}</dd></div>
          {unit.fxRateToDisplay ? (
            <div><dt>{t('unit.fxToDisplay')}</dt><dd>{unit.fxRateToDisplay.toFixed(6)} {purchaseCurrency}/{currency}</dd></div>
          ) : null}
          <div><dt>{t('unit.dealer')}</dt><dd>{unit.dealer || t('common.emDash')}</dd></div>
          <div><dt>{t('unit.storage')}</dt><dd>{unit.storageLocation || t('common.emDash')}</dd></div>
          <div><dt>{t('unit.condition')}</dt><dd>{unit.condition || t('common.emDash')}</dd></div>
          <div><dt>{t('unit.mintageYear')}</dt><dd>{unit.mintageYear || t('common.emDash')}</dd></div>
          <div><dt>{t('unit.daysHeld')}</dt><dd>{unit.daysHeld || t('common.emDash')}</dd></div>
          <div>
            <dt>{t('unit.annualized')}</dt>
            <dd className={profitClass(unit.annualizedReturnPct || 0)}>
              {unit.daysHeld ? formatPercent(unit.annualizedReturnPct || 0) : t('common.emDash')}
              {unit.isRealized ? ` ${t('unit.realizedSuffix')}` : ` ${t('unit.unrealizedSuffix')}`}
            </dd>
          </div>
          {unit.status === 'held' && (
            <div>
              <dt>{t('unit.breakEven')}</dt>
              <dd>{formatMoney(breakEven, currency)} / {unitLabel}</dd>
            </div>
          )}
          <div><dt>{t('unit.notes')}</dt><dd>{unit.notes || t('common.emDash')}</dd></div>
          <div>
            <dt>{t('add.purity')}</dt>
            <dd>{unit.purity}{isUnusualPurity(unit.purity) ? ` ${t('unit.unusual')}` : ''}</dd>
          </div>
        </dl>
      </section>

      <AttachmentsPanel
        ownerType="unit"
        ownerId={unit.id}
        kind="photo"
        title={t('unit.photos')}
        emptyLabel={t('unit.noPhotos')}
        addLabel={t('unit.addPhoto')}
      />
      <AttachmentsPanel
        ownerType="investment"
        ownerId={unit.investmentId}
        kind="receipt"
        title={t('unit.receipts')}
        emptyLabel={t('unit.noReceipts')}
        addLabel={t('unit.addReceipt')}
      />

      {error && <p className="error-text">{error}</p>}

      {showSell && (
        <Modal onBackdropClick={() => setShowSell(false)}>
          <form onSubmit={handleSell}>
            <h2>{t('unit.sellTitle')}</h2>
            <p className="muted">{t('unit.sellBody')}</p>
            <label>
              {t('unit.saleDate')}
              <input type="date" value={soldAt} onChange={(event) => setSoldAt(event.target.value)} required />
            </label>
            <label>
              {t('unit.salePrice')} ({purchaseCurrency})
              <input
                type="number"
                min="0"
                step="any"
                value={salePrice}
                onChange={(event) => setSalePrice(Number(event.target.value))}
                required
              />
            </label>
            <div className="button-row">
              <button type="button" className="btn btn-ghost" onClick={() => setShowSell(false)}>{t('common.cancel')}</button>
              <button type="submit" className="btn btn-primary" disabled={busy}>
                {busy ? t('add.saving') : t('unit.confirmSale')}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {showEdit && (
        <Modal
          onBackdropClick={() => setShowEdit(false)}
          panelClassName="modal-panel modal-panel-wide"
        >
          <form onSubmit={handleEdit}>
            <h2>{t('unit.editTitle')}</h2>
            <div className="form-grid">
              <label>
                {t('add.metal')}
                <select
                  value={editForm.metal}
                  onChange={(event) => setEditForm({...editForm, metal: event.target.value as MetalSymbol})}
                >
                  <option value="XAU">{t('common.gold')}</option>
                  <option value="XAG">{t('common.silver')}</option>
                </select>
              </label>
              <label>
                {t('add.form')}
                <select
                  value={editForm.form}
                  onChange={(event) => setEditForm({...editForm, form: event.target.value as Form})}
                >
                  <option value="bar">{t('common.bar')}</option>
                  <option value="coin">{t('common.coin')}</option>
                  <option value="other">{t('common.other')}</option>
                </select>
              </label>
              <label>
                {t('add.weight')}
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={editForm.weight}
                  onChange={(event) => setEditForm({...editForm, weight: Number(event.target.value)})}
                  required
                />
              </label>
              <label>
                {t('add.unit')}
                <select
                  value={editForm.weightUnit}
                  onChange={(event) => setEditForm({...editForm, weightUnit: event.target.value as WeightUnit})}
                >
                  <option value="g">{t('common.grams')}</option>
                  <option value="troy_oz">{t('common.troyOz')}</option>
                  <option value="kg">{t('common.kilograms')}</option>
                </select>
              </label>
              <label>
                {t('add.purity')}
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={editForm.purity}
                  onChange={(event) => setEditForm({...editForm, purity: Number(event.target.value)})}
                />
                {isUnusualPurity(editForm.purity) && (
                  <span className="field-warning">{t('add.unusualPurity')}</span>
                )}
              </label>
              <label>
                {t('add.purchaseDate')}
                <input
                  type="date"
                  value={editForm.purchasedAt}
                  onChange={(event) => setEditForm({...editForm, purchasedAt: event.target.value})}
                  required
                />
              </label>
              <label>
                {t('add.brand')}
                <input
                  value={editForm.brand}
                  onChange={(event) => setEditForm({...editForm, brand: event.target.value})}
                />
              </label>
              <label>
                {t('add.productName')}
                <input
                  value={editForm.productName}
                  onChange={(event) => setEditForm({...editForm, productName: event.target.value})}
                />
              </label>
              <label className={`gift-toggle span-2${editForm.isGift ? ' is-active' : ''}`}>
                <input
                  type="checkbox"
                  checked={editForm.isGift}
                  onChange={(event) => setEditForm({
                    ...editForm,
                    isGift: event.target.checked,
                    purchasePrice: event.target.checked ? 0 : editForm.purchasePrice,
                  })}
                />
                <span className="gift-toggle-copy">
                  <strong>{t('unit.isGift')}</strong>
                  <span>{t('unit.giftHelp')}</span>
                </span>
              </label>
              <label>
                {t('unit.purchasePrice')}
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={editForm.isGift ? 0 : editForm.purchasePrice}
                  onChange={(event) => setEditForm({...editForm, purchasePrice: Number(event.target.value)})}
                  required={!editForm.isGift}
                  disabled={editForm.isGift}
                />
              </label>
              <label>
                {t('unit.spotAtPurchase')}
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={editForm.spotWorthAtPurchase}
                  onChange={(event) => setEditForm({...editForm, spotWorthAtPurchase: Number(event.target.value)})}
                  required
                />
              </label>
              <label>
                {t('unit.status')}
                <select
                  value={editForm.status}
                  onChange={(event) => setEditForm({...editForm, status: event.target.value as UnitStatus})}
                >
                  <option value="held">{t('unit.statusHeld')}</option>
                  <option value="sold">{t('unit.statusSold')}</option>
                </select>
              </label>
              <label>
                {t('unit.dealer')}
                <input
                  value={editForm.dealer}
                  onChange={(event) => setEditForm({...editForm, dealer: event.target.value})}
                />
              </label>
              <label>
                {t('unit.storage')}
                <input
                  value={editForm.storageLocation}
                  onChange={(event) => setEditForm({...editForm, storageLocation: event.target.value})}
                />
              </label>
              <label>
                {t('unit.condition')}
                <input
                  value={editForm.condition}
                  onChange={(event) => setEditForm({...editForm, condition: event.target.value})}
                />
              </label>
              <label>
                {t('unit.mintageYear')}
                <input
                  type="number"
                  min="0"
                  step="1"
                  value={editForm.mintageYear || ''}
                  onChange={(event) => setEditForm({
                    ...editForm,
                    mintageYear: Number(event.target.value) || 0,
                  })}
                />
              </label>
              {editForm.status === 'sold' && (
                <>
                  <label>
                    {t('unit.saleDate')}
                    <input
                      type="date"
                      value={editForm.soldAt}
                      onChange={(event) => setEditForm({...editForm, soldAt: event.target.value})}
                      required
                    />
                  </label>
                  <label>
                    {t('unit.salePrice')}
                    <input
                      type="number"
                      min="0"
                      step="any"
                      value={editForm.salePrice}
                      onChange={(event) => setEditForm({...editForm, salePrice: Number(event.target.value)})}
                      required
                    />
                  </label>
                </>
              )}
              <label className="span-2">
                {t('unit.notes')}
                <input
                  value={editForm.notes}
                  onChange={(event) => setEditForm({...editForm, notes: event.target.value})}
                />
              </label>
            </div>
            <div className="button-row">
              <button type="button" className="btn btn-ghost" onClick={() => setShowEdit(false)}>{t('common.cancel')}</button>
              <button type="submit" className="btn btn-primary" disabled={busy}>
                {busy ? t('add.saving') : t('unit.saveChanges')}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {showDelete && (
        <ConfirmDialog
          title={t('unit.deleteTitle')}
          message={t('unit.deleteBody')}
          confirmLabel={t('unit.deleteAction')}
          danger
          busy={busy}
          onCancel={() => setShowDelete(false)}
          onConfirm={handleDelete}
        />
      )}
    </div>
  );
}
