import {FormEvent, useEffect, useState} from 'react';
import {
  ChangePIN,
  CheckForUpdates,
  CreateBackup,
  GetAppVersion,
  GetSettings,
  InstallUpdate,
  ListRecentlyDeleted,
  PurgeHoldingUnit,
  RestoreBackup,
  RestoreHoldingUnit,
  SaveRecoveryKit,
  UpdateSettings,
  VerifyBackup,
} from '../../wailsjs/go/main/App';
import {ConfirmDialog} from '../components/ConfirmDialog';
import {isCompletePin, PinInput} from '../components/PinInput';
import {showToast} from '../components/Toast';
import {UpdateDialog} from '../components/UpdateDialog';
import {useLocale} from '../i18n/LocaleContext';
import {
  AppSettings,
  BackupManifest,
  BackupVerifyResult,
  Currency,
  PriceSource,
  SpotPriceUnit,
  UITheme,
  UnitValuation,
  UpdateCheckResult,
} from '../types';
import {formatAppError, formatUserMessage, isUserCancelled} from '../utils/errors';
import {formatDate, formatWeight, formLabel, metalLabel} from '../utils/format';
import {applyTheme} from '../utils/theme';
import {subscribeUpdateDownloadProgress, UpdateDownloadProgress} from '../utils/updateProgress';

interface SettingsProps {
  onInventoryChanged?: () => void;
  onVaultLocked?: () => void;
}

export function Settings({onInventoryChanged, onVaultLocked}: SettingsProps) {
  const {t} = useLocale();
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [currentPin, setCurrentPin] = useState('');
  const [newPin, setNewPin] = useState('');
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [pinState, setPinState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [showConfirmPinChange, setShowConfirmPinChange] = useState(false);
  const [error, setError] = useState('');
  const [pinError, setPinError] = useState('');
  const [deletedUnits, setDeletedUnits] = useState<UnitValuation[]>([]);
  const [deletedError, setDeletedError] = useState('');
  const [deletedBusyId, setDeletedBusyId] = useState('');
  const [purgeTargetId, setPurgeTargetId] = useState('');
  const [backupBusy, setBackupBusy] = useState(false);
  const [appVersion, setAppVersion] = useState('dev');
  const [updateCheck, setUpdateCheck] = useState<UpdateCheckResult | null>(null);
  const [updateBusy, setUpdateBusy] = useState(false);
  const [updateMessage, setUpdateMessage] = useState('');
  const [updateError, setUpdateError] = useState('');
  const [showUpdateConfirm, setShowUpdateConfirm] = useState(false);
  const [updateProgress, setUpdateProgress] = useState<UpdateDownloadProgress | null>(null);
  const [backupMessage, setBackupMessage] = useState('');
  const [backupError, setBackupError] = useState('');
  const [backupPrompt, setBackupPrompt] = useState<'backup' | 'verify' | 'restore' | 'kit' | null>(null);
  const [promptPin, setPromptPin] = useState('');
  const [promptRecoveryKey, setPromptRecoveryKey] = useState('');

  function loadDeleted() {
    ListRecentlyDeleted()
      .then((result) => setDeletedUnits((result || []) as UnitValuation[]))
      .catch((err) => setDeletedError(formatAppError(err)));
  }

  useEffect(() => {
    GetSettings()
      .then((result) => {
        const loaded = result as AppSettings;
        setSettings({
          ...loaded,
          spotPriceUnit: loaded.spotPriceUnit || 'troy_oz',
          uiTheme: loaded.uiTheme || 'dark',
          skippedUpdateVersion: loaded.skippedUpdateVersion || '',
        });
      })
      .catch((err) => setError(formatAppError(err)));
    GetAppVersion()
      .then((value) => setAppVersion(value || 'dev'))
      .catch(() => setAppVersion('dev'));
    loadDeleted();
  }, []);

  async function handleCheckForUpdates() {
    setUpdateBusy(true);
    setUpdateError('');
    setUpdateMessage('');
    try {
      const result = await CheckForUpdates() as UpdateCheckResult;
      setUpdateCheck(result);
      if (result?.available) {
        showToast({message: t('settings.updateAvailable', {version: result.latestVersion})});
      } else {
        showToast({message: t('settings.upToDate')});
      }
    } catch (err) {
      setUpdateError(formatAppError(err));
    } finally {
      setUpdateBusy(false);
    }
  }

  async function handleInstallUpdate() {
    setUpdateBusy(true);
    setUpdateError('');
    setUpdateProgress(null);
    const unsubscribe = subscribeUpdateDownloadProgress(setUpdateProgress);
    try {
      await InstallUpdate();
    } catch (err) {
      setUpdateError(formatAppError(err));
      setUpdateBusy(false);
      setUpdateProgress(null);
      setShowUpdateConfirm(false);
    } finally {
      unsubscribe();
    }
  }

  async function handleSkipUpdate() {
    if (!settings || !updateCheck?.available || updateBusy) return;
    setUpdateError('');
    try {
      const nextSettings: AppSettings = {
        ...settings,
        skippedUpdateVersion: updateCheck.latestVersion,
      };
      await UpdateSettings(nextSettings as never);
      setSettings(nextSettings);
      setShowUpdateConfirm(false);
    } catch (err) {
      setUpdateError(formatAppError(err));
    }
  }

  useEffect(() => {
    if (saveState !== 'saved') return;
    const timer = window.setTimeout(() => setSaveState('idle'), 2200);
    return () => window.clearTimeout(timer);
  }, [saveState]);

  useEffect(() => {
    if (pinState !== 'saved') return;
    const timer = window.setTimeout(() => setPinState('idle'), 2200);
    return () => window.clearTimeout(timer);
  }, [pinState]);

  async function handleSave(event: FormEvent) {
    event.preventDefault();
    if (!settings) return;
    setSaveState('saving');
    setError('');
    try {
      await UpdateSettings(settings as never);
      applyTheme(settings.uiTheme);
      setSaveState('saved');
    } catch (err) {
      setError(formatAppError(err));
      setSaveState('error');
    }
  }

  async function submitChangePin() {
    if (pinState === 'saving') return;
    setPinState('saving');
    setPinError('');
    if (!isCompletePin(currentPin) || !isCompletePin(newPin)) {
      setShowConfirmPinChange(false);
      setPinError(t('lock.pinExact'));
      setPinState('error');
      return;
    }
    try {
      await ChangePIN(currentPin, newPin);
      setCurrentPin('');
      setNewPin('');
      setShowConfirmPinChange(false);
      setPinState('saved');
    } catch (err) {
      setShowConfirmPinChange(false);
      setPinError(formatAppError(err));
      setPinState('error');
    }
  }

  function handleChangePin(event: FormEvent) {
    event.preventDefault();
    if (pinState === 'saving') return;
    if (!isCompletePin(currentPin) || !isCompletePin(newPin)) {
      setPinError(t('lock.pinExact'));
      setPinState('error');
      return;
    }
    setPinError('');
    setPinState('idle');
    setShowConfirmPinChange(true);
  }

  function openBackupPrompt(prompt: 'backup' | 'verify' | 'restore' | 'kit') {
    setBackupError('');
    setBackupMessage('');
    setPromptPin('');
    setPromptRecoveryKey('');
    setBackupPrompt(prompt);
  }

  function closeBackupPrompt() {
    if (backupBusy) return;
    setBackupPrompt(null);
    setPromptPin('');
    setPromptRecoveryKey('');
  }

  async function submitBackupPrompt() {
    if (backupBusy || !backupPrompt) return;

    const needsPin = backupPrompt === 'backup' || backupPrompt === 'kit';
    const needsRecovery = backupPrompt === 'verify' || backupPrompt === 'restore';
    if (needsPin && !isCompletePin(promptPin)) {
      setBackupError(t('lock.pinExact'));
      return;
    }
    if (needsRecovery && !promptRecoveryKey.trim()) {
      setBackupError(t('settings.recoveryRequired'));
      return;
    }

    setBackupBusy(true);
    setBackupError('');
    setBackupMessage('');
    try {
      if (backupPrompt === 'backup') {
        const manifest = await CreateBackup(promptPin) as BackupManifest;
        setBackupMessage(
          t('settings.backupSaved', {
            units: manifest.unitCount,
            attachments: manifest.attachmentCount,
          }),
        );
      } else if (backupPrompt === 'kit') {
        await SaveRecoveryKit(promptPin);
        setBackupMessage(t('settings.recoveryKitSaved'));
      } else if (backupPrompt === 'verify') {
        const result = await VerifyBackup(promptRecoveryKey.trim()) as BackupVerifyResult;
        if (result.valid) {
          setBackupMessage(
            result.message || t('settings.verifyOk', {
              units: result.unitCount,
              attachments: result.attachmentCount,
            }),
          );
        } else {
          setBackupError(formatUserMessage(result.message || t('settings.verifyFailed')));
          return;
        }
      } else {
        await RestoreBackup(promptRecoveryKey.trim(), true);
        setBackupPrompt(null);
        onVaultLocked?.();
        return;
      }
      setBackupPrompt(null);
      setPromptPin('');
      setPromptRecoveryKey('');
    } catch (err) {
      if (isUserCancelled(err)) {
        closeBackupPrompt();
        return;
      }
      setBackupError(formatAppError(err));
    } finally {
      setBackupBusy(false);
    }
  }

  if (!settings) {
    return <div className="glass panel"><p className="muted">{t('dashboard.loading')}</p></div>;
  }

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('settings.eyebrow')}</p>
          <h1>{t('settings.title')}</h1>
        </div>
      </header>

      <form className="glass panel form-panel" onSubmit={handleSave}>
        <h2>{t('settings.portfolio')}</h2>
        <div className="form-grid">
          <label>
            {t('settings.displayCurrency')}
            <select
              value={settings.displayCurrency}
              onChange={(event) => setSettings({
                ...settings,
                displayCurrency: event.target.value as Currency,
              })}
            >
              <option value="EUR">EUR</option>
              <option value="USD">USD</option>
              <option value="CHF">CHF</option>
            </select>
          </label>
          <label>
            {t('settings.spotUnit')}
            <select
              value={settings.spotPriceUnit || 'troy_oz'}
              onChange={(event) => setSettings({
                ...settings,
                spotPriceUnit: event.target.value as SpotPriceUnit,
              })}
            >
              <option value="g">{t('common.gram')}</option>
              <option value="troy_oz">{t('common.troyOz')}</option>
              <option value="kilogram">{t('common.kilogram')}</option>
            </select>
          </label>
          <label>
            {t('settings.appearance')}
            <select
              value={settings.uiTheme || 'dark'}
              onChange={(event) => {
                const uiTheme = event.target.value as UITheme;
                setSettings({...settings, uiTheme});
                applyTheme(uiTheme);
              }}
            >
              <option value="dark">{t('common.dark')}</option>
              <option value="light">{t('common.light')}</option>
            </select>
          </label>
          <label>
            {t('settings.autoLock')}
            <input
              type="number"
              min="1"
              value={settings.autoLockMinutes}
              onChange={(event) => setSettings({
                ...settings,
                autoLockMinutes: Number(event.target.value),
              })}
            />
          </label>
          <label>
            {t('settings.priceSource')}
            <select
              value={settings.priceSource}
              onChange={(event) => setSettings({
                ...settings,
                priceSource: event.target.value as PriceSource,
              })}
            >
              <option value="metalpriceapi">{t('settings.priceSourceMetalprice')}</option>
              <option value="middleman">{t('settings.priceSourceMiddleman')}</option>
            </select>
          </label>
          <label className="span-2">
            {t('settings.apiKey')}
            <input
              type="password"
              value={settings.metalpriceApiKey}
              onChange={(event) => setSettings({
                ...settings,
                metalpriceApiKey: event.target.value,
              })}
              placeholder={t('settings.apiKeyPlaceholder')}
            />
          </label>
          <label className="span-2">
            {t('settings.middleman')}
            <input
              value={settings.middlemanBaseUrl}
              onChange={(event) => {
                const middlemanBaseUrl = event.target.value;
                setSettings({
                  ...settings,
                  middlemanBaseUrl,
                  // Filling a middleman URL selects that source so quotes actually use it.
                  priceSource: middlemanBaseUrl.trim()
                    ? 'middleman'
                    : settings.priceSource,
                });
              }}
              placeholder={t('settings.middlemanPlaceholder')}
            />
          </label>
        </div>
        <div className="form-actions">
          <button
            className={`btn btn-primary ${saveState === 'saved' ? 'btn-success' : ''}`}
            type="submit"
            disabled={saveState === 'saving'}
          >
            {saveState === 'saving' ? t('settings.saving') : saveState === 'saved' ? t('settings.saved') : t('settings.save')}
          </button>
          {saveState === 'saved' && <span className="inline-feedback success-text">{t('settings.saved')}</span>}
          {saveState === 'error' && error && <span className="inline-feedback error-text">{error}</span>}
        </div>
      </form>

      <section className="glass panel form-panel">
        <h2>{t('settings.recentlyDeleted')}</h2>
        <p className="muted small">{t('settings.recentlyDeletedBody')}</p>
        {deletedError && <p className="error-text">{deletedError}</p>}
        {deletedUnits.length === 0 && !deletedError && (
          <p className="muted">{t('settings.recentlyDeletedEmpty')}</p>
        )}
        <div className="deleted-list">
          {deletedUnits.map((unit) => (
            <div key={unit.id} className="deleted-row">
              <div>
                <strong>
                  {unit.productName || `${metalLabel(unit.metal)} ${formLabel(unit.form)}`}
                </strong>
                <p className="muted small">
                  {formatWeight(unit.weightGrams)} - {t('settings.boughtDeleted', {
                    bought: formatDate(unit.purchasedAt),
                    deleted: unit.deletedAt ? formatDate(unit.deletedAt) : t('common.emDash'),
                  })}
                </p>
              </div>
              <div className="button-row">
                <button
                  type="button"
                  className="btn btn-secondary"
                  disabled={deletedBusyId === unit.id}
                  onClick={() => {
                    setDeletedBusyId(unit.id);
                    setDeletedError('');
                    RestoreHoldingUnit(unit.id)
                      .then(() => {
                        loadDeleted();
                        onInventoryChanged?.();
                      })
                      .catch((err) => setDeletedError(formatAppError(err)))
                      .finally(() => setDeletedBusyId(''));
                  }}
                >
                  {t('settings.restore')}
                </button>
                <button
                  type="button"
                  className="btn btn-ghost"
                  disabled={deletedBusyId === unit.id}
                  onClick={() => setPurgeTargetId(unit.id)}
                >
                  {t('settings.purgeAction')}
                </button>
              </div>
            </div>
          ))}
        </div>
      </section>

      <form className="glass panel form-panel" onSubmit={handleChangePin}>
        <h2>{t('settings.changePin')}</h2>
        <div className="form-grid">
          <label className="span-2">
            {t('settings.currentPin')}
            <PinInput
              id="settings-current-pin"
              value={currentPin}
              onChange={setCurrentPin}
              onComplete={() => {
                document.getElementById('settings-new-pin')?.focus();
              }}
              hasError={pinState === 'error'}
              ariaLabel={t('settings.currentPin')}
              compact
            />
          </label>
          <label className="span-2">
            {t('settings.newPin')}
            <PinInput
              id="settings-new-pin"
              value={newPin}
              onChange={setNewPin}
              hasError={pinState === 'error'}
              ariaLabel={t('settings.newPin')}
              compact
            />
          </label>
        </div>
        <div className="form-actions">
          <button
            className={`btn btn-primary ${pinState === 'saved' ? 'btn-success' : ''}`}
            type="submit"
            disabled={pinState === 'saving' || !isCompletePin(currentPin) || !isCompletePin(newPin)}
          >
            {pinState === 'saving'
              ? t('settings.updatingPin')
              : pinState === 'saved'
                ? t('settings.pinUpdated')
                : t('settings.updatePin')}
          </button>
          {pinState === 'error' && pinError && <span className="inline-feedback error-text">{pinError}</span>}
        </div>
      </form>

      <section className="glass panel form-panel">
        <h2>{t('settings.backup')}</h2>
        <p className="muted small">{t('settings.backupBody')}</p>
        <div className="form-actions button-row">
          <button
            className="btn btn-primary"
            type="button"
            disabled={backupBusy}
            onClick={() => openBackupPrompt('backup')}
          >
            {t('settings.createBackup')}
          </button>
          <button
            className="btn btn-secondary"
            type="button"
            disabled={backupBusy}
            onClick={() => openBackupPrompt('verify')}
          >
            {t('settings.verifyBackup')}
          </button>
          <button
            className="btn btn-secondary"
            type="button"
            disabled={backupBusy}
            onClick={() => openBackupPrompt('restore')}
          >
            {t('settings.restoreBackup')}
          </button>
          <button
            className="btn btn-ghost"
            type="button"
            disabled={backupBusy}
            onClick={() => openBackupPrompt('kit')}
          >
            {t('settings.saveRecoveryKit')}
          </button>
        </div>
        {backupMessage && <p className="success-text">{backupMessage}</p>}
        {backupError && !backupPrompt && <p className="error-text">{backupError}</p>}

        <h3>{t('settings.migrateTitle')}</h3>
        <p className="muted small">{t('settings.migrateBody')}</p>
      </section>

      <section className="glass panel form-panel">
        <h2>{t('settings.about')}</h2>
        <p className="muted small">{t('settings.aboutBody')}</p>
        <p><strong>{t('settings.version', {version: appVersion})}</strong></p>
        {updateCheck?.available && updateCheck.releaseNotes && (
          <pre className="release-notes muted small">{updateCheck.releaseNotes}</pre>
        )}
        <div className="form-actions button-row">
          <button
            className="btn btn-secondary"
            type="button"
            disabled={updateBusy}
            onClick={handleCheckForUpdates}
          >
            {updateBusy && !showUpdateConfirm ? t('settings.checkingUpdates') : t('settings.checkUpdates')}
          </button>
          {updateCheck?.available && (
            <button
              className="btn btn-primary"
              type="button"
              disabled={updateBusy}
              onClick={() => setShowUpdateConfirm(true)}
            >
              {t('settings.installUpdate')}
            </button>
          )}
        </div>
        {updateMessage && <p className="success-text">{updateMessage}</p>}
        {updateError && !showUpdateConfirm && <p className="error-text">{updateError}</p>}
      </section>

      {showUpdateConfirm && updateCheck?.available && (
        <UpdateDialog
          version={updateCheck.latestVersion}
          releaseNotes={updateCheck.releaseNotes}
          busy={updateBusy}
          progress={updateProgress}
          error={updateError}
          onLater={() => {
            if (!updateBusy) setShowUpdateConfirm(false);
          }}
          onSkip={() => void handleSkipUpdate()}
          onInstall={() => void handleInstallUpdate()}
        />
      )}

      {showConfirmPinChange && (
        <ConfirmDialog
          title={t('settings.confirmPinTitle')}
          message={t('settings.confirmPinBody')}
          confirmLabel={t('settings.confirmPinAction')}
          cancelLabel={t('common.cancel')}
          busy={pinState === 'saving'}
          onCancel={() => {
            if (pinState === 'saving') return;
            setShowConfirmPinChange(false);
          }}
          onConfirm={() => {
            void submitChangePin();
          }}
        />
      )}

      {backupPrompt && (
        <ConfirmDialog
          title={
            backupPrompt === 'backup' ? t('settings.createBackup')
              : backupPrompt === 'verify' ? t('settings.verifyBackup')
                : backupPrompt === 'restore' ? t('settings.restoreBackup')
                  : t('settings.saveRecoveryKit')
          }
          message={
            backupPrompt === 'backup' || backupPrompt === 'kit'
              ? t('settings.backupEnterPin')
              : backupPrompt === 'restore'
                ? t('settings.restoreConfirmBody')
                : t('settings.verifyConfirmBody')
          }
          confirmLabel={
            backupPrompt === 'restore' ? t('settings.restoreBackup') : t('common.continue')
          }
          cancelLabel={t('common.cancel')}
          danger={backupPrompt === 'restore'}
          busy={backupBusy}
          panelClassName={
            backupPrompt === 'backup' || backupPrompt === 'kit'
              ? 'modal-panel modal-panel-pin'
              : 'modal-panel'
          }
          onCancel={closeBackupPrompt}
          onConfirm={() => {
            void submitBackupPrompt();
          }}
        >
          {(backupPrompt === 'backup' || backupPrompt === 'kit') && (
            <PinInput
              id="settings-backup-prompt-pin"
              value={promptPin}
              onChange={setPromptPin}
              hasError={Boolean(backupError)}
              ariaLabel={t('lock.pinLabel')}
              compact
              autoFocus
            />
          )}
          {(backupPrompt === 'verify' || backupPrompt === 'restore') && (
            <label>
              {t('settings.backupEnterRecovery')}
              <textarea
                rows={3}
                value={promptRecoveryKey}
                onChange={(event) => setPromptRecoveryKey(event.target.value)}
                disabled={backupBusy}
                autoFocus
              />
            </label>
          )}
          {backupError && <p className="error-text">{backupError}</p>}
        </ConfirmDialog>
      )}

      {purgeTargetId && (
        <ConfirmDialog
          title={t('settings.purgeTitle')}
          message={t('settings.purgeBody')}
          confirmLabel={t('settings.purgeAction')}
          cancelLabel={t('common.cancel')}
          danger
          busy={deletedBusyId === purgeTargetId}
          onCancel={() => {
            if (deletedBusyId) return;
            setPurgeTargetId('');
          }}
          onConfirm={() => {
            const unitId = purgeTargetId;
            setDeletedBusyId(unitId);
            setDeletedError('');
            PurgeHoldingUnit(unitId)
              .then(() => {
                setPurgeTargetId('');
                loadDeleted();
                onInventoryChanged?.();
              })
              .catch((err) => setDeletedError(formatAppError(err)))
              .finally(() => setDeletedBusyId(''));
          }}
        />
      )}
    </div>
  );
}
