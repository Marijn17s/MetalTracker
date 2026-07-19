import {ReactNode} from 'react';
import {useLocale} from '../i18n/LocaleContext';
import {Modal} from './Modal';

interface ConfirmDialogProps {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  busy?: boolean;
  panelClassName?: string;
  onConfirm: () => void;
  onCancel: () => void;
  children?: ReactNode;
}

export function ConfirmDialog({
  title,
  message,
  confirmLabel,
  cancelLabel,
  danger = false,
  busy = false,
  panelClassName,
  onConfirm,
  onCancel,
  children,
}: ConfirmDialogProps) {
  const {t} = useLocale();
  const resolvedConfirmLabel = confirmLabel ?? t('confirm.default');
  const resolvedCancelLabel = cancelLabel ?? t('common.cancel');

  return (
    <Modal
      onBackdropClick={busy ? undefined : onCancel}
      panelClassName={panelClassName || 'modal-panel'}
    >
      <h2>{title}</h2>
      <p className="muted">{message}</p>
      {children}
      <div className="button-row">
        <button type="button" className="btn btn-ghost" onClick={onCancel} disabled={busy}>
          {resolvedCancelLabel}
        </button>
        <button
          type="button"
          className={`btn ${danger ? 'btn-danger' : 'btn-primary'}`}
          onClick={onConfirm}
          disabled={busy}
        >
          {busy ? t('confirm.working') : resolvedConfirmLabel}
        </button>
      </div>
    </Modal>
  );
}
