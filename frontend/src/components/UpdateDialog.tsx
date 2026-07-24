import {useLocale} from '../i18n/LocaleContext';
import {UpdateDownloadProgress} from '../utils/updateProgress';
import {Modal} from './Modal';

interface UpdateDialogProps {
  version: string;
  releaseNotes?: string;
  busy: boolean;
  progress: UpdateDownloadProgress | null;
  error?: string;
  onLater: () => void;
  onSkip: () => void;
  onInstall: () => void;
}

export function UpdateDialog({
  version,
  releaseNotes,
  busy,
  progress,
  error,
  onLater,
  onSkip,
  onInstall,
}: UpdateDialogProps) {
  const {t} = useLocale();
  const hasKnownTotal = progress !== null && progress.total > 0 && progress.percent >= 0;
  const percent = hasKnownTotal ? Math.min(100, Math.round(progress.percent)) : null;

  return (
    <Modal onBackdropClick={busy ? undefined : onLater} panelClassName="modal-panel modal-panel-update">
      <h2>{t('settings.updateConfirmTitle')}</h2>
      <p className="muted">{t('settings.updateConfirmBody', {version})}</p>
      {releaseNotes ? <pre className="release-notes muted small">{releaseNotes}</pre> : null}
      {busy ? (
        <div className="update-progress" aria-live="polite">
          <div
            className={`update-progress-track${hasKnownTotal ? '' : ' is-indeterminate'}`}
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={percent ?? undefined}
          >
            <div
              className="update-progress-fill"
              style={hasKnownTotal ? {width: `${percent}%`} : undefined}
            />
          </div>
          <p className="muted small">
            {percent !== null
              ? t('settings.updateProgress', {percent})
              : t('settings.updateProgressIndeterminate')}
          </p>
        </div>
      ) : null}
      {error ? <p className="error-text">{error}</p> : null}
      <div className="button-row update-dialog-actions">
        <button type="button" className="btn btn-ghost" onClick={onLater} disabled={busy}>
          {t('settings.updateLater')}
        </button>
        <button type="button" className="btn btn-secondary" onClick={onSkip} disabled={busy}>
          {t('settings.updateSkip')}
        </button>
        <button type="button" className="btn btn-primary" onClick={onInstall} disabled={busy}>
          {busy ? t('settings.installingUpdate') : t('settings.updateConfirmAction')}
        </button>
      </div>
    </Modal>
  );
}