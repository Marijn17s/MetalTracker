import {useCallback, useEffect, useState} from 'react';
import {
  CheckForUpdates,
  GetDateFormatPattern,
  GetSettings,
  InstallUpdate,
  IsUnlocked,
  Lock,
  RestoreHoldingUnit,
  TouchActivity,
  UpdateSettings,
} from '../wailsjs/go/main/App';
import {LockScreen} from './components/LockScreen';
import {Shell} from './components/Shell';
import {showToast, ToastHost} from './components/Toast';
import {UpdateDialog} from './components/UpdateDialog';
import {LocaleProvider, useLocale} from './i18n/LocaleContext';
import {AddInvestment} from './pages/AddInvestment';
import {Dashboard} from './pages/Dashboard';
import {GroupDetail} from './pages/GroupDetail';
import {Help} from './pages/Help';
import {Holdings} from './pages/Holdings';
import {Monthly} from './pages/Monthly';
import {Settings} from './pages/Settings';
import {SoldArchive} from './pages/SoldArchive';
import {UnitDetail} from './pages/UnitDetail';
import {AddInvestmentPrefill, AppSettings, AppView, UpdateCheckResult} from './types';
import {formatAppError} from './utils/errors';
import {setDateFormatPattern, setFormatLocale} from './utils/format';
import {applyTheme} from './utils/theme';
import {subscribeUpdateDownloadProgress, UpdateDownloadProgress} from './utils/updateProgress';
import './App.css';

function AppShell() {
  const {intl, t} = useLocale();
  const [unlocked, setUnlocked] = useState(false);
  const [checking, setChecking] = useState(true);
  const [view, setView] = useState<AppView>('dashboard');
  const [selectedProductKey, setSelectedProductKey] = useState('');
  const [selectedUnitId, setSelectedUnitId] = useState('');
  const [holdingsRefresh, setHoldingsRefresh] = useState(0);
  const [addPrefill, setAddPrefill] = useState<AddInvestmentPrefill | null>(null);
  const [unitBackView, setUnitBackView] = useState<AppView>('group');
  const [appSettings, setAppSettings] = useState<AppSettings | null>(null);
  const [startupUpdate, setStartupUpdate] = useState<UpdateCheckResult | null>(null);
  const [showStartupUpdate, setShowStartupUpdate] = useState(false);
  const [startupUpdateBusy, setStartupUpdateBusy] = useState(false);
  const [startupUpdateProgress, setStartupUpdateProgress] = useState<UpdateDownloadProgress | null>(null);
  const [startupUpdateError, setStartupUpdateError] = useState('');

  useEffect(() => {
    setFormatLocale(intl);
  }, [intl]);

  useEffect(() => {
    IsUnlocked()
      .then((value) => setUnlocked(value))
      .finally(() => setChecking(false));
    GetDateFormatPattern()
      .then(setDateFormatPattern)
      .catch(() => setDateFormatPattern(''));
  }, []);

  useEffect(() => {
    if (!unlocked) {
      applyTheme('dark');
      setAppSettings(null);
      setStartupUpdate(null);
      setShowStartupUpdate(false);
      setStartupUpdateBusy(false);
      setStartupUpdateProgress(null);
      setStartupUpdateError('');
      return;
    }

    let cancelled = false;
    GetSettings()
      .then(async (settings) => {
        const loaded = settings as AppSettings;
        if (cancelled) return;
        applyTheme(loaded.uiTheme);
        setAppSettings({
          ...loaded,
          skippedUpdateVersion: loaded.skippedUpdateVersion || '',
        });
        try {
          const result = await CheckForUpdates() as UpdateCheckResult;
          if (cancelled || !result?.available) return;
          if ((loaded.skippedUpdateVersion || '').trim() === result.latestVersion) return;
          setStartupUpdate(result);
          setShowStartupUpdate(true);
        } catch {
          // Startup checks are silent so network errors do not interrupt the vault.
        }
      })
      .catch(() => applyTheme('dark'));

    return () => {
      cancelled = true;
    };
  }, [unlocked]);

  useEffect(() => {
    if (!unlocked) return;
    const onActivity = () => {
      TouchActivity().catch(() => undefined);
    };
    window.addEventListener('pointerdown', onActivity);
    window.addEventListener('keydown', onActivity);
    const interval = window.setInterval(() => {
      IsUnlocked().then((value) => {
        if (!value) {
          setUnlocked(false);
        }
      }).catch(() => setUnlocked(false));
    }, 15000);
    return () => {
      window.removeEventListener('pointerdown', onActivity);
      window.removeEventListener('keydown', onActivity);
      window.clearInterval(interval);
    };
  }, [unlocked]);

  const handleLock = useCallback(async () => {
    await Lock();
    setUnlocked(false);
    setView('dashboard');
  }, []);

  const clearAddPrefill = useCallback(() => {
    setAddPrefill(null);
  }, []);

  const handleStartupInstallUpdate = useCallback(async () => {
    setStartupUpdateBusy(true);
    setStartupUpdateError('');
    setStartupUpdateProgress(null);
    const unsubscribe = subscribeUpdateDownloadProgress(setStartupUpdateProgress);
    try {
      await InstallUpdate();
    } catch (err) {
      setStartupUpdateError(formatAppError(err));
      setStartupUpdateBusy(false);
      setStartupUpdateProgress(null);
    } finally {
      unsubscribe();
    }
  }, []);

  const handleStartupSkipUpdate = useCallback(async () => {
    if (!appSettings || !startupUpdate?.available || startupUpdateBusy) return;
    setStartupUpdateError('');
    try {
      const nextSettings: AppSettings = {
        ...appSettings,
        skippedUpdateVersion: startupUpdate.latestVersion,
      };
      await UpdateSettings(nextSettings as never);
      setAppSettings(nextSettings);
      setShowStartupUpdate(false);
    } catch (err) {
      setStartupUpdateError(formatAppError(err));
    }
  }, [appSettings, startupUpdate, startupUpdateBusy]);

  useEffect(() => {
    if (!unlocked) return;
    function onKeyDown(event: KeyboardEvent) {
      if (!(event.ctrlKey || event.metaKey)) return;
      if (event.key.toLowerCase() !== 'l') return;
      event.preventDefault();
      void handleLock();
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [unlocked, handleLock]);

  if (checking) {
    return (
      <div className="lock-screen">
        <div className="glass lock-panel">
          <p className="brand-mark">{t('lock.brand')}</p>
          <p className="muted">{t('lock.starting')}</p>
        </div>
      </div>
    );
  }

  if (!unlocked) {
    return <LockScreen onUnlocked={() => {
      setUnlocked(true);
      setView('dashboard');
    }} />;
  }

  return (
    <>
      <Shell
        view={view}
        holdingsStackActive={view === 'group' || (view === 'unit' && unitBackView === 'group')}
        soldStackActive={view === 'sold' || (view === 'unit' && unitBackView === 'sold')}
        onNavigate={(next) => {
          setView(next);
          if (next === 'holdings' || next === 'sold') {
            setSelectedProductKey('');
            setSelectedUnitId('');
          }
        }}
        onLock={handleLock}
      >
        {view === 'dashboard' && (
          <Dashboard
            key={`dash-${holdingsRefresh}`}
            onNavigate={setView}
          />
        )}
        {view === 'holdings' && (
          <Holdings
            key={`hold-${holdingsRefresh}`}
            onOpenGroup={(productKey) => {
              setSelectedProductKey(productKey);
              setView('group');
            }}
          />
        )}
        {view === 'sold' && (
          <SoldArchive
            key={`sold-${holdingsRefresh}`}
            onOpenUnit={(unitId, productKey) => {
              setSelectedProductKey(productKey);
              setSelectedUnitId(unitId);
              setUnitBackView('sold');
              setView('unit');
            }}
          />
        )}
        {view === 'group' && (
          <GroupDetail
            key={`group-${holdingsRefresh}-${selectedProductKey}`}
            productKey={selectedProductKey}
            onBack={() => setView('holdings')}
            onOpenUnit={(unitId) => {
              setSelectedUnitId(unitId);
              setUnitBackView('group');
              setView('unit');
            }}
            onAddMore={(prefill) => {
              setAddPrefill(prefill);
              setView('add');
            }}
            onChanged={() => setHoldingsRefresh((value) => value + 1)}
          />
        )}
        {view === 'unit' && (
          <UnitDetail
            unitId={selectedUnitId}
            onBack={() => setView(unitBackView)}
            onChanged={() => setHoldingsRefresh((value) => value + 1)}
            onDeleted={(deletedUnitId) => {
              setHoldingsRefresh((value) => value + 1);
              setSelectedUnitId('');
              setView(unitBackView === 'sold' ? 'sold' : 'group');
              showToast({
                message: t('toast.deleted'),
                actionLabel: t('toast.undo'),
                onAction: async () => {
                  await RestoreHoldingUnit(deletedUnitId);
                  setHoldingsRefresh((value) => value + 1);
                },
              });
            }}
            onAddMore={(prefill) => {
              setAddPrefill(prefill);
              setView('add');
            }}
          />
        )}
        {view === 'add' && (
          <AddInvestment
            onCreated={() => setHoldingsRefresh((value) => value + 1)}
            prefill={addPrefill}
            onPrefillConsumed={clearAddPrefill}
          />
        )}
        {view === 'monthly' && <Monthly />}
        {view === 'settings' && (
          <Settings
            onInventoryChanged={() => setHoldingsRefresh((value) => value + 1)}
            onVaultLocked={() => setUnlocked(false)}
          />
        )}
        {view === 'help' && <Help />}
      </Shell>
      <ToastHost />
      {showStartupUpdate && startupUpdate?.available && (
        <UpdateDialog
          version={startupUpdate.latestVersion}
          releaseNotes={startupUpdate.releaseNotes}
          busy={startupUpdateBusy}
          progress={startupUpdateProgress}
          error={startupUpdateError}
          onLater={() => {
            if (!startupUpdateBusy) setShowStartupUpdate(false);
          }}
          onSkip={() => void handleStartupSkipUpdate()}
          onInstall={() => void handleStartupInstallUpdate()}
        />
      )}
    </>
  );
}

function App() {
  return (
    <LocaleProvider>
      <AppShell />
    </LocaleProvider>
  );
}

export default App;
