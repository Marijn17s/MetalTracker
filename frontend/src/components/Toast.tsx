import {useEffect, useState} from 'react';
import {createPortal} from 'react-dom';
import {useLocale} from '../i18n/LocaleContext';

export interface ToastRequest {
  message: string;
  actionLabel?: string;
  onAction?: () => void | Promise<void>;
  durationMs?: number;
}

interface ActiveToast extends ToastRequest {
  id: number;
}

let showToastHandler: ((request: ToastRequest) => void) | null = null;
let toastSequence = 0;

export function showToast(request: ToastRequest): void {
  showToastHandler?.(request);
}

export function ToastHost() {
  const {t} = useLocale();
  const [toast, setToast] = useState<ActiveToast | null>(null);

  useEffect(() => {
    showToastHandler = (request) => {
      toastSequence += 1;
      setToast({...request, id: toastSequence});
    };
    return () => {
      showToastHandler = null;
    };
  }, []);

  useEffect(() => {
    if (!toast) return;
    const durationMs = toast.durationMs ?? 8000;
    const timer = window.setTimeout(() => setToast(null), durationMs);
    return () => window.clearTimeout(timer);
  }, [toast]);

  if (!toast) return null;

  return createPortal(
    <div className="toast-host" role="status">
      <div className="toast glass">
        <span>{toast.message}</span>
        <div className="toast-actions">
          {toast.actionLabel && toast.onAction && (
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => {
                void Promise.resolve(toast.onAction?.()).finally(() => setToast(null));
              }}
            >
              {toast.actionLabel}
            </button>
          )}
          <button type="button" className="btn btn-ghost" onClick={() => setToast(null)}>
            {t('toast.dismiss')}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
