import {FormEvent, useEffect, useRef, useState} from 'react';
import {Check, Copy} from 'lucide-react';
import {
  RecoverWithKey,
  SaveRecoveryKitFromKey,
  SetupVault,
  Unlock,
  VaultExists,
} from '../../wailsjs/go/main/App';
import {useLocale} from '../i18n/LocaleContext';
import {formatAppError, isUserCancelled} from '../utils/errors';
import {isCompletePin, PinInput} from './PinInput';

interface LockScreenProps {
  onUnlocked: () => void;
}

type Mode = 'loading' | 'setup' | 'unlock' | 'recover' | 'recoveryShown';

export function LockScreen({onUnlocked}: LockScreenProps) {
  const {t} = useLocale();
  const [mode, setMode] = useState<Mode>('loading');
  const [pin, setPin] = useState('');
  const [confirmPin, setConfirmPin] = useState('');
  const [recoveryKey, setRecoveryKey] = useState('');
  const [newPin, setNewPin] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [shownRecoveryKey, setShownRecoveryKey] = useState('');
  const [copied, setCopied] = useState(false);
  const confirmPinContainerRef = useRef<HTMLLabelElement | null>(null);
  const busyRef = useRef(false);
  const pinRef = useRef('');
  const recoveryKeyRef = useRef('');

  useEffect(() => {
    VaultExists()
      .then((exists) => setMode(exists ? 'unlock' : 'setup'))
      .catch(() => setMode('setup'));
  }, []);

  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), 2000);
    return () => window.clearTimeout(timer);
  }, [copied]);

  useEffect(() => {
    busyRef.current = busy;
  }, [busy]);

  useEffect(() => {
    pinRef.current = pin;
  }, [pin]);

  useEffect(() => {
    recoveryKeyRef.current = recoveryKey;
  }, [recoveryKey]);

  async function submitSetup(setupPin: string, setupConfirmPin: string) {
    if (busyRef.current) return;
    setError('');
    if (!isCompletePin(setupPin) || !isCompletePin(setupConfirmPin)) {
      setError(t('lock.pinExact'));
      return;
    }
    if (setupPin !== setupConfirmPin) {
      setError(t('lock.pinMismatch'));
      return;
    }
    setBusy(true);
    try {
      const result = await SetupVault(setupPin);
      setShownRecoveryKey(result.recoveryKey);
      setMode('recoveryShown');
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  async function submitUnlock(unlockPin: string) {
    if (busyRef.current) return;
    setError('');
    if (!isCompletePin(unlockPin)) {
      setError(t('lock.pinExact'));
      return;
    }
    setBusy(true);
    try {
      await Unlock(unlockPin);
      onUnlocked();
    } catch (err) {
      setError(formatAppError(err));
      setPin('');
    } finally {
      setBusy(false);
    }
  }

  async function submitRecover(recoverPin: string, key: string) {
    if (busyRef.current) return;
    setError('');
    if (!isCompletePin(recoverPin)) {
      setError(t('lock.pinExact'));
      return;
    }
    if (!key.trim()) {
      return;
    }
    setBusy(true);
    try {
      await RecoverWithKey(key.trim(), recoverPin);
      onUnlocked();
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleSetup(event: FormEvent) {
    event.preventDefault();
    await submitSetup(pin, confirmPin);
  }

  async function handleUnlock(event: FormEvent) {
    event.preventDefault();
    await submitUnlock(pin);
  }

  async function handleRecover(event: FormEvent) {
    event.preventDefault();
    await submitRecover(newPin, recoveryKey);
  }

  async function handleCopyRecoveryKey() {
    try {
      await navigator.clipboard.writeText(shownRecoveryKey);
      setCopied(true);
    } catch {
      setError(t('lock.copyFailed'));
    }
  }

  if (mode === 'loading') {
    return (
      <div className="lock-screen">
        <div className="glass lock-panel fade-in">
          <p className="brand-mark">{t('lock.brand')}</p>
          <p className="muted">{t('lock.opening')}</p>
        </div>
      </div>
    );
  }

  if (mode === 'recoveryShown') {
    return (
      <div className="lock-screen">
        <div className="glass lock-panel fade-in">
          <p className="brand-mark">{t('lock.brand')}</p>
          <h1>{t('lock.saveRecoveryTitle')}</h1>
          <p className="muted">{t('lock.saveRecoveryBody')}</p>
          <div className="recovery-key-row">
            <div className="recovery-key">{shownRecoveryKey}</div>
            <button
              className={`btn copy-btn ${copied ? 'btn-success' : 'btn-ghost'}`}
              type="button"
              onClick={handleCopyRecoveryKey}
              aria-label={copied ? t('lock.copied') : t('lock.copy')}
              title={copied ? t('lock.copied') : t('lock.copy')}
            >
              {copied ? <Check size={18} strokeWidth={2.25} aria-hidden="true" /> : <Copy size={18} strokeWidth={2.25} aria-hidden="true" />}
            </button>
          </div>
          <div className="stack lock-actions">
            <button
              className="btn btn-ghost"
              type="button"
              onClick={() => {
                void SaveRecoveryKitFromKey(shownRecoveryKey).catch((err) => {
                  if (!isUserCancelled(err)) {
                    setError(formatAppError(err));
                  }
                });
              }}
            >
              {t('lock.saveRecoveryKit')}
            </button>
            {error && <p className="error-text">{error}</p>}
            <button className="btn btn-primary" type="button" onClick={onUnlocked}>
              {t('lock.continue')}
            </button>
          </div>
        </div>
      </div>
    );
  }

  const title =
    mode === 'setup' ? t('lock.setupTitle') :
      mode === 'recover' ? t('lock.recoverTitle') :
        t('lock.unlockTitle');
  const body =
    mode === 'setup' ? t('lock.setupBody') :
      mode === 'recover' ? t('lock.recoverBody') :
        t('lock.unlockBody');

  return (
    <div className="lock-screen">
      <div className="glass lock-panel fade-in">
        <p className="brand-mark">{t('lock.brand')}</p>
        <h1>{title}</h1>
        <p className="muted">{body}</p>

        {mode === 'setup' && (
          <form className="stack" onSubmit={handleSetup}>
            <label>
              {t('lock.pinLabel')}
              <PinInput
                id="setup-pin"
                value={pin}
                onChange={setPin}
                onComplete={() => {
                  const confirmInput = confirmPinContainerRef.current?.querySelector('input');
                  confirmInput?.focus();
                }}
                disabled={busy}
                autoFocus
                hasError={Boolean(error)}
                ariaLabel={t('lock.pinLabel')}
              />
            </label>
            <label ref={confirmPinContainerRef}>
              {t('lock.confirmPinLabel')}
              <PinInput
                id="setup-confirm-pin"
                value={confirmPin}
                onChange={setConfirmPin}
                onComplete={(completedConfirm) => {
                  void submitSetup(pinRef.current, completedConfirm);
                }}
                disabled={busy}
                hasError={Boolean(error)}
                ariaLabel={t('lock.confirmPinLabel')}
              />
            </label>
            <button className="btn btn-primary" type="submit" disabled={busy || !isCompletePin(pin) || !isCompletePin(confirmPin)}>
              {t('lock.createVault')}
            </button>
          </form>
        )}

        {mode === 'unlock' && (
          <form className="stack" onSubmit={handleUnlock}>
            <label>
              {t('lock.pinLabel')}
              <PinInput
                id="unlock-pin"
                value={pin}
                onChange={setPin}
                onComplete={(completedPin) => {
                  void submitUnlock(completedPin);
                }}
                disabled={busy}
                autoFocus
                hasError={Boolean(error)}
                ariaLabel={t('lock.pinLabel')}
              />
            </label>
            <button className="btn btn-primary" type="submit" disabled={busy || !isCompletePin(pin)}>
              {t('lock.unlock')}
            </button>
            <button
              className="btn btn-ghost"
              type="button"
              onClick={() => {
                setError('');
                setMode('recover');
              }}
            >
              {t('lock.forgotPin')}
            </button>
          </form>
        )}

        {mode === 'recover' && (
          <form className="stack" onSubmit={handleRecover}>
            <label>
              {t('lock.recoveryKey')}
              <textarea
                rows={3}
                value={recoveryKey}
                onChange={(event) => setRecoveryKey(event.target.value)}
              />
            </label>
            <label>
              {t('lock.newPinLabel')}
              <PinInput
                id="recover-pin"
                value={newPin}
                onChange={setNewPin}
                onComplete={(completedPin) => {
                  void submitRecover(completedPin, recoveryKeyRef.current);
                }}
                disabled={busy}
                hasError={Boolean(error)}
                ariaLabel={t('lock.newPinLabel')}
              />
            </label>
            <button className="btn btn-primary" type="submit" disabled={busy || !isCompletePin(newPin)}>
              {t('lock.restoreVault')}
            </button>
            <button
              className="btn btn-ghost"
              type="button"
              onClick={() => {
                setError('');
                setMode('unlock');
              }}
            >
              {t('lock.backToUnlock')}
            </button>
          </form>
        )}

        {error && <p className="error-text">{error}</p>}
      </div>
    </div>
  );
}
