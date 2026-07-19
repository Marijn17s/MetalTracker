import {MouseEvent, useEffect, useState} from 'react';
import {
  AddAttachment,
  DeleteAttachment,
  GetAttachmentBytes,
  ListAttachments,
  SaveAttachment,
} from '../../wailsjs/go/main/App';
import {useLocale} from '../i18n/LocaleContext';
import {Attachment, AttachmentKind, AttachmentOwnerType} from '../types';
import {formatAppError, isUserCancelled} from '../utils/errors';
import {Modal} from './Modal';

interface AttachmentsPanelProps {
  ownerType: AttachmentOwnerType;
  ownerId: string;
  kind: AttachmentKind;
  title: string;
  emptyLabel: string;
  addLabel: string;
}

interface PreviewState {
  attachment: Attachment;
  objectUrl: string;
}

function SaveIcon() {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
      <path
        fill="currentColor"
        d="M17 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V7zm-1 16H6V5h9.17L18 7.83V19zm-5-1a3 3 0 1 0 0-6 3 3 0 0 0 0 6zm-4-9h8V5H7z"
      />
    </svg>
  );
}

function DeleteIcon() {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
      <path
        fill="currentColor"
        d="M9 3h6l1 1h4v2H4V4h4zm1 6h2v8h-2zm4 0h2v8h-2zM6 8h12l-1 12H7z"
      />
    </svg>
  );
}

function FileIcon() {
  return (
    <svg viewBox="0 0 24 24" width="28" height="28" aria-hidden="true">
      <path
        fill="currentColor"
        d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zm0 2.5L18.5 9H14zM8 13h8v2H8zm0 4h8v2H8z"
      />
    </svg>
  );
}

export function AttachmentsPanel({
  ownerType,
  ownerId,
  kind,
  title,
  emptyLabel,
  addLabel,
}: AttachmentsPanelProps) {
  const {t} = useLocale();
  const [items, setItems] = useState<Attachment[]>([]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState<PreviewState | null>(null);
  const [thumbnails, setThumbnails] = useState<Record<string, string>>({});

  async function refresh() {
    try {
      const result = await ListAttachments(ownerType, ownerId);
      setItems((result || []) as Attachment[]);
    } catch (err) {
      setError(formatAppError(err));
    }
  }

  useEffect(() => {
    void refresh();
  }, [ownerType, ownerId]);

  useEffect(() => {
    let cancelled = false;
    const imageItems = items.filter((item) => item.contentType.startsWith('image/'));
    Promise.all(imageItems.map(async (item) => {
      try {
        const payload = await GetAttachmentBytes(item.id);
        return {
          id: item.id,
          url: `data:${payload.contentType};base64,${payload.dataBase64}`,
        };
      } catch {
        return null;
      }
    })).then((loaded) => {
      if (cancelled) return;
      const next: Record<string, string> = {};
      for (const entry of loaded) {
        if (entry) next[entry.id] = entry.url;
      }
      setThumbnails(next);
    });
    return () => {
      cancelled = true;
    };
  }, [items]);

  async function handleAdd() {
    setBusy(true);
    setError('');
    try {
      await AddAttachment(ownerType, ownerId, kind);
      await refresh();
    } catch (err) {
      if (!isUserCancelled(err)) {
        setError(formatAppError(err));
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleOpen(attachment: Attachment) {
    setBusy(true);
    setError('');
    try {
      const payload = await GetAttachmentBytes(attachment.id);
      const objectUrl = `data:${payload.contentType};base64,${payload.dataBase64}`;
      setPreview({attachment, objectUrl});
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleSave(attachmentId: string, event?: MouseEvent) {
    event?.stopPropagation();
    setBusy(true);
    setError('');
    try {
      await SaveAttachment(attachmentId);
    } catch (err) {
      if (!isUserCancelled(err)) {
        setError(formatAppError(err));
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete(attachmentId: string, event?: MouseEvent) {
    event?.stopPropagation();
    setBusy(true);
    setError('');
    try {
      await DeleteAttachment(attachmentId);
      if (preview?.attachment.id === attachmentId) {
        setPreview(null);
      }
      await refresh();
    } catch (err) {
      setError(formatAppError(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="glass panel">
      <div className="attachments-header">
        <h2>{title}</h2>
        <button type="button" className="btn btn-ghost" onClick={() => void handleAdd()} disabled={busy}>
          {addLabel}
        </button>
      </div>

      {items.length === 0 && <p className="muted">{emptyLabel}</p>}

      {items.length > 0 && (
        <div className="attachment-grid">
          {items.map((item) => {
            const isImage = item.contentType.startsWith('image/');
            const thumbnail = thumbnails[item.id];
            return (
              <article key={item.id} className="attachment-card">
                <div
                  className={`attachment-tile ${isImage && thumbnail ? 'has-image' : 'is-file'}`}
                  role="button"
                  tabIndex={0}
                  aria-label={t('attachments.open', {name: item.filename})}
                  onClick={() => void handleOpen(item)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      void handleOpen(item);
                    }
                  }}
                >
                  {isImage && thumbnail ? (
                    <img src={thumbnail} alt="" />
                  ) : (
                    <div className="attachment-file-face">
                      <FileIcon />
                      <span>{item.filename}</span>
                    </div>
                  )}
                  <div className="attachment-tile-overlay" aria-hidden="true" />
                  <div className="attachment-tile-actions">
                    <button
                      type="button"
                      className="attachment-icon-btn"
                      title={t('attachments.saveAs')}
                      aria-label={t('attachments.saveAsNamed', {name: item.filename})}
                      disabled={busy}
                      onClick={(event) => void handleSave(item.id, event)}
                    >
                      <SaveIcon />
                    </button>
                    <button
                      type="button"
                      className="attachment-icon-btn danger"
                      title={t('attachments.delete')}
                      aria-label={t('attachments.deleteNamed', {name: item.filename})}
                      disabled={busy}
                      onClick={(event) => void handleDelete(item.id, event)}
                    >
                      <DeleteIcon />
                    </button>
                  </div>
                </div>
              </article>
            );
          })}
        </div>
      )}

      {error && <p className="error-text">{error}</p>}

      {preview && (
        <Modal
          onBackdropClick={() => setPreview(null)}
          panelClassName="modal-panel attachment-preview-modal"
        >
          <div className="attachment-preview-header">
            <h2>{preview.attachment.filename}</h2>
            <button type="button" className="btn btn-ghost" onClick={() => setPreview(null)}>
              {t('attachments.close')}
            </button>
          </div>
          {preview.attachment.contentType.startsWith('image/') ? (
            <img
              className="attachment-preview"
              src={preview.objectUrl}
              alt={preview.attachment.filename}
            />
          ) : (
            <div className="attachment-preview-fallback">
              <FileIcon />
              <p className="muted">{t('attachments.previewFallback')}</p>
            </div>
          )}
          <div className="button-row">
            <button
              type="button"
              className="btn btn-ghost btn-danger-text"
              onClick={() => void handleDelete(preview.attachment.id)}
              disabled={busy}
            >
              {t('attachments.delete')}
            </button>
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => void handleSave(preview.attachment.id)}
              disabled={busy}
            >
              {t('attachments.saveAsEllipsis')}
            </button>
          </div>
        </Modal>
      )}
    </section>
  );
}
