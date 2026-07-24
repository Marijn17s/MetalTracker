import {EventsOn} from '../../wailsjs/runtime/runtime';

export interface UpdateDownloadProgress {
  downloaded: number;
  total: number;
  percent: number;
}

export function subscribeUpdateDownloadProgress(
  onProgress: (progress: UpdateDownloadProgress) => void,
): () => void {
  return EventsOn('update:download-progress', (payload: unknown) => {
    if (!payload || typeof payload !== 'object') return;
    const record = payload as Record<string, unknown>;
    const downloaded = Number(record.downloaded);
    if (!Number.isFinite(downloaded)) return;
    const total = Number(record.total);
    const percent = Number(record.percent);
    onProgress({
      downloaded,
      total: Number.isFinite(total) ? total : -1,
      percent: Number.isFinite(percent) ? percent : -1,
    });
  });
}