import {FormEvent, KeyboardEvent, useEffect, useRef, useState} from 'react';
import {CreateInvestment, ListDealerSummaries} from '../../wailsjs/go/main/App';
import {showToast} from '../components/Toast';
import {useLocale} from '../i18n/LocaleContext';
import {
  AddInvestmentPrefill,
  CreateInvestmentRequest,
  Currency,
  DealerSummary,
  Form,
  InvestmentLineInput,
  MetalSymbol,
  WeightUnit,
} from '../types';
import {formatAppError} from '../utils/errors';
import {formatMoney, isUnusualPurity, todayISO} from '../utils/format';

interface AddInvestmentProps {
  onCreated: () => void;
  prefill?: AddInvestmentPrefill | null;
  onPrefillConsumed?: () => void;
}

type NumericInputValue = number | '';

type InvestmentLineForm = Omit<
  InvestmentLineInput,
  'weight' | 'purity' | 'quantity' | 'totalPurchasePrice' | 'mintageYear'
> & {
  weight: NumericInputValue;
  purity: NumericInputValue;
  quantity: NumericInputValue;
  totalPurchasePrice: NumericInputValue;
  mintageYear: NumericInputValue;
};

function numericInputValue(value: string): NumericInputValue {
  return value === '' ? '' : Number(value);
}

function emptyLine(): InvestmentLineForm {
  return {
    assetClass: 'precious_metal',
    metal: 'XAU',
    form: 'bar',
    weight: 1,
    weightUnit: 'g',
    purity: 999.9,
    brand: '',
    productName: '',
    quantity: 1,
    totalPurchasePrice: 0,
    totalSpotWorth: 0,
    spotWorthProvided: false,
    isGift: false,
    storageLocation: '',
    condition: '',
    mintageYear: 0,
  };
}

function lineFromPrefill(prefill: AddInvestmentPrefill): InvestmentLineForm {
  return {
    ...emptyLine(),
    metal: prefill.metal,
    form: prefill.form,
    weight: prefill.weightGrams,
    weightUnit: 'g',
    purity: prefill.purity,
    brand: prefill.brand,
    productName: prefill.productName,
    condition: prefill.condition || '',
    mintageYear: prefill.mintageYear || 0,
  };
}

function FieldLabel({children, required = false}: {children: string; required?: boolean}) {
  return (
    <span>
      {children}
      {required ? <span className="required-mark" aria-hidden="true"> *</span> : null}
    </span>
  );
}

export function AddInvestment({onCreated, prefill, onPrefillConsumed}: AddInvestmentProps) {
  const {t} = useLocale();
  const [purchasedAt, setPurchasedAt] = useState(todayISO());
  const [currency, setCurrency] = useState<Currency>('EUR');
  const [dealer, setDealer] = useState('');
  const [notes, setNotes] = useState('');
  const [lines, setLines] = useState<InvestmentLineForm[]>([emptyLine()]);
  const [dealers, setDealers] = useState<DealerSummary[]>([]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const purchaseDateRef = useRef<HTMLInputElement | null>(null);
  const formRef = useRef<HTMLFormElement | null>(null);

  useEffect(() => {
    ListDealerSummaries()
      .then((result) => setDealers((result || []) as DealerSummary[]))
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    purchaseDateRef.current?.focus();
  }, []);

  useEffect(() => {
    if (!prefill) return;
    setLines([lineFromPrefill(prefill)]);
    const successMessage = t('add.prefilled');
    showToast({message: successMessage});
    onPrefillConsumed?.();
  }, [prefill, onPrefillConsumed, t]);

  function updateLine(index: number, patch: Partial<InvestmentLineForm>) {
    setLines((current) => current.map((line, lineIndex) => (
      lineIndex === index ? {...line, ...patch} : line
    )));
  }

  function addLine() {
    setLines((current) => [...current, emptyLine()]);
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setError('');
    setBusy(true);

    const request: CreateInvestmentRequest = {
      purchasedAt,
      currency,
      dealer,
      notes,
      lines: lines.map((line) => ({
        ...line,
        weight: Number(line.weight),
        purity: Number(line.purity),
        quantity: Number(line.quantity),
        totalPurchasePrice: line.isGift ? 0 : Number(line.totalPurchasePrice),
        mintageYear: line.mintageYear === '' ? 0 : Number(line.mintageYear),
        totalSpotWorth: 0,
        spotWorthProvided: false,
      })),
    };

    try {
      await CreateInvestment(request as never);
      const successMessage = t('add.saved');
      showToast({message: successMessage});
      setLines([emptyLine()]);
      ListDealerSummaries()
        .then((result) => setDealers((result || []) as DealerSummary[]))
        .catch(() => undefined);
      onCreated();
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  function handleFormKeyDown(event: KeyboardEvent<HTMLFormElement>) {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      event.preventDefault();
      formRef.current?.requestSubmit();
      return;
    }
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'l') {
      // Shell uses Ctrl+L for lock; don't steal it for add-line.
      return;
    }
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLowerCase() === 'a') {
      event.preventDefault();
      addLine();
    }
  }

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('add.eyebrow')}</p>
          <h1>{t('add.title')}</h1>
          <p className="muted small">
            {t('add.subtitle')}
          </p>
        </div>
      </header>

      <form
        ref={formRef}
        className="glass panel form-panel"
        onSubmit={handleSubmit}
        onKeyDown={handleFormKeyDown}
      >
        <div className="form-grid">
          <label>
            <FieldLabel required>{t('add.purchaseDate')}</FieldLabel>
            <input
              ref={purchaseDateRef}
              type="date"
              value={purchasedAt}
              onChange={(event) => setPurchasedAt(event.target.value)}
              required
              enterKeyHint="next"
            />
          </label>
          <label>
            <FieldLabel required>{t('add.currency')}</FieldLabel>
            <select
              value={currency}
              onChange={(event) => setCurrency(event.target.value as Currency)}
              required
            >
              <option value="EUR">EUR</option>
              <option value="USD">USD</option>
              <option value="CHF">CHF</option>
            </select>
          </label>
          <label>
            <FieldLabel>{t('add.dealer')}</FieldLabel>
            <input
              list="dealer-directory"
              value={dealer}
              onChange={(event) => setDealer(event.target.value)}
              placeholder={t('add.optional')}
              enterKeyHint="next"
            />
            <datalist id="dealer-directory">
              {dealers.map((item) => (
                <option key={item.name} value={item.name} />
              ))}
            </datalist>
          </label>
          <label className="span-2">
            <FieldLabel>{t('add.notes')}</FieldLabel>
            <input
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
              placeholder={t('add.optional')}
              enterKeyHint="next"
            />
          </label>
        </div>

        {dealers.length > 0 && (
          <div className="dealer-stats">
            <p className="muted small">{t('add.previousDealers')}</p>
            <div className="dealer-stat-list">
              {dealers.map((item) => (
                <button
                  key={item.name}
                  type="button"
                  className="dealer-stat-chip"
                  onClick={() => setDealer(item.name)}
                >
                  <strong>{item.name}</strong>
                  <span>
                    {t('add.dealerUnits', {count: item.unitCount})} - {formatMoney(item.totalSpent, item.currency)}
                  </span>
                </button>
              ))}
            </div>
          </div>
        )}

        <div className="lines-header">
          <h2>{t('add.productsTitle')}</h2>
        </div>

        {lines.map((line, index) => (
          <div className="glass-inset line-block" key={index}>
            <div className="line-title">
              <h3>{t('add.line', {n: index + 1})}</h3>
              {lines.length > 1 && (
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => setLines((current) => current.filter((_, lineIndex) => lineIndex !== index))}
                >
                  {t('add.remove')}
                </button>
              )}
            </div>
            <div className="form-grid">
              <label>
                <FieldLabel required>{t('add.metal')}</FieldLabel>
                <select
                  value={line.metal}
                  onChange={(event) => updateLine(index, {metal: event.target.value as MetalSymbol})}
                  required
                >
                  <option value="XAU">{t('common.gold')}</option>
                  <option value="XAG">{t('common.silver')}</option>
                </select>
              </label>
              <label>
                <FieldLabel required>{t('add.form')}</FieldLabel>
                <select
                  value={line.form}
                  onChange={(event) => updateLine(index, {form: event.target.value as Form})}
                  required
                >
                  <option value="bar">{t('common.bar')}</option>
                  <option value="coin">{t('common.coin')}</option>
                  <option value="other">{t('common.other')}</option>
                </select>
              </label>
              <label>
                <FieldLabel required>{t('add.weight')}</FieldLabel>
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={line.weight}
                  onChange={(event) => updateLine(index, {weight: numericInputValue(event.target.value)})}
                  required
                  enterKeyHint="next"
                />
              </label>
              <label>
                <FieldLabel required>{t('add.unit')}</FieldLabel>
                <select
                  value={line.weightUnit}
                  onChange={(event) => updateLine(index, {weightUnit: event.target.value as WeightUnit})}
                  required
                >
                  <option value="g">{t('common.grams')}</option>
                  <option value="troy_oz">{t('common.troyOz')}</option>
                  <option value="kg">{t('common.kilograms')}</option>
                </select>
              </label>
              <label>
                <FieldLabel required>{t('add.purity')}</FieldLabel>
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={line.purity}
                  onChange={(event) => updateLine(index, {purity: numericInputValue(event.target.value)})}
                  required
                  enterKeyHint="next"
                />
                {line.purity !== '' && isUnusualPurity(line.purity) && (
                  <span className="field-warning">{t('add.unusualPurity')}</span>
                )}
              </label>
              <label>
                <FieldLabel required>{t('add.quantity')}</FieldLabel>
                <input
                  type="number"
                  min="1"
                  step="1"
                  value={line.quantity}
                  onChange={(event) => updateLine(index, {quantity: numericInputValue(event.target.value)})}
                  required
                  enterKeyHint="next"
                />
              </label>
              <label className={`gift-toggle span-2${line.isGift ? ' is-active' : ''}`}>
                <input
                  type="checkbox"
                  checked={line.isGift}
                  onChange={(event) => updateLine(index, {
                    isGift: event.target.checked,
                    totalPurchasePrice: event.target.checked ? 0 : line.totalPurchasePrice,
                  })}
                />
                <span className="gift-toggle-copy">
                  <strong>{t('add.isGift')}</strong>
                  <span>{t('add.giftHelp')}</span>
                </span>
              </label>
              <label>
                <FieldLabel required={!line.isGift}>{t('add.totalPurchasePrice')}</FieldLabel>
                <input
                  type="number"
                  min="0"
                  step="any"
                  value={line.isGift ? 0 : line.totalPurchasePrice}
                  onChange={(event) => updateLine(index, {
                    totalPurchasePrice: numericInputValue(event.target.value),
                  })}
                  required={!line.isGift}
                  disabled={line.isGift}
                  enterKeyHint={index === lines.length - 1 ? 'done' : 'next'}
                />
              </label>
              <label>
                <FieldLabel>{t('add.brand')}</FieldLabel>
                <input
                  value={line.brand}
                  onChange={(event) => updateLine(index, {brand: event.target.value})}
                  placeholder={t('add.optional')}
                  enterKeyHint="next"
                />
              </label>
              <label>
                <FieldLabel>{t('add.productName')}</FieldLabel>
                <input
                  value={line.productName}
                  onChange={(event) => updateLine(index, {productName: event.target.value})}
                  placeholder={t('add.optional')}
                  enterKeyHint="next"
                />
              </label>
              <label>
                <FieldLabel>{t('add.storage')}</FieldLabel>
                <input
                  value={line.storageLocation}
                  onChange={(event) => updateLine(index, {storageLocation: event.target.value})}
                  placeholder={t('add.optional')}
                  enterKeyHint="next"
                />
              </label>
              <label>
                <FieldLabel>{t('add.condition')}</FieldLabel>
                <input
                  list="condition-suggestions"
                  value={line.condition}
                  onChange={(event) => updateLine(index, {condition: event.target.value})}
                  placeholder={t('add.optional')}
                  enterKeyHint="next"
                />
              </label>
              <label>
                <FieldLabel>{t('add.mintageYear')}</FieldLabel>
                <input
                  type="number"
                  min="0"
                  step="1"
                  value={line.mintageYear}
                  onChange={(event) => updateLine(index, {
                    mintageYear: numericInputValue(event.target.value),
                  })}
                  placeholder={t('add.optional')}
                  enterKeyHint="done"
                />
              </label>
            </div>
            <p className="muted small">
              {t('add.createsUnits', {count: line.quantity})}
            </p>
          </div>
        ))}

        <datalist id="condition-suggestions">
          <option value={t('add.conditionBU')} />
          <option value={t('add.conditionProof')} />
          <option value={t('add.conditionUNC')} />
          <option value={t('add.conditionCirculated')} />
        </datalist>

        {error && <p className="error-text">{error}</p>}

        <div className="sticky-form-actions">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={addLine}
          >
            {t('add.addLine')}
          </button>
          <button className="btn btn-primary" type="submit" disabled={busy}>
            {busy ? t('add.saving') : t('add.save')}
          </button>
        </div>
      </form>
    </div>
  );
}
