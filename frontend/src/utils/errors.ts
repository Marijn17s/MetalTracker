import {translate, TranslationKey} from '../i18n/translations';

const KNOWN_CODES = new Set([
  'invalid_pin',
  'vault_locked',
  'vault_exists',
  'vault_missing',
  'weak_pin',
  'invalid_recovery',
  'price_unavailable',
  'price_stale',
  'invalid_api_key',
  'not_found',
  'validation',
  'cancelled',
  'internal',
  'update_unavailable',
]);

const CODE_KEYS: Record<string, TranslationKey> = {
  invalid_pin: 'errors.invalid_pin',
  vault_locked: 'errors.vault_locked',
  vault_exists: 'errors.vault_exists',
  vault_missing: 'errors.vault_missing',
  weak_pin: 'errors.weak_pin',
  invalid_recovery: 'errors.invalid_recovery',
  price_unavailable: 'errors.price_unavailable',
  price_stale: 'errors.price_stale',
  invalid_api_key: 'errors.invalid_api_key',
  not_found: 'errors.not_found',
  validation: 'errors.validation',
  internal: 'errors.internal',
  update_unavailable: 'errors.update_unavailable',
};

function codeMessage(code: string): string {
  const key = CODE_KEYS[code] || CODE_KEYS.internal;
  return translate(key);
}

function normalizeErrorText(error: unknown): string {
  if (error instanceof Error) {
    return error.message.trim();
  }
  if (typeof error === 'string') {
    return error.trim();
  }
  if (error && typeof error === 'object' && 'message' in error) {
    const message = (error as {message?: unknown}).message;
    if (typeof message === 'string' && message.trim()) {
      return message.trim();
    }
  }
  return String(error ?? '').trim();
}

function stripErrorPrefix(text: string): string {
  let current = text;
  while (/^error:\s*/i.test(current)) {
    current = current.replace(/^error:\s*/i, '').trim();
  }
  return current;
}

function parseAppError(text: string): {code: string; message: string} | null {
  const cleaned = stripErrorPrefix(text);
  const separator = cleaned.indexOf(': ');
  if (separator <= 0) {
    return null;
  }
  const code = cleaned.slice(0, separator).trim();
  const message = cleaned.slice(separator + 2).trim();
  if (!KNOWN_CODES.has(code)) {
    return null;
  }
  return {code, message};
}

function looksLikeUserCancelled(message: string): boolean {
  const lower = message.trim().toLowerCase();
  if (!lower) {
    return false;
  }
  return (
    lower === 'cancelled' ||
    lower === 'canceled' ||
    lower.endsWith(' cancelled') ||
    lower.endsWith(' canceled') ||
    lower.includes('cancelled.') ||
    lower.includes('canceled.') ||
    lower === 'no file selected' ||
    lower === 'no file selected.' ||
    lower === 'no save location selected' ||
    lower === 'no save location selected.' ||
    lower.startsWith('backup cancelled') ||
    lower.startsWith('verify cancelled') ||
    lower.startsWith('restore cancelled') ||
    lower.startsWith('save cancelled')
  );
}

/** True when a validation message looks like a raw/system error rather than UI copy. */
function looksTechnical(message: string): boolean {
  const text = message.trim();
  if (!text) {
    return true;
  }
  const lower = text.toLowerCase();
  if (text.length > 140) {
    return true;
  }
  return (
    lower.includes('http://') ||
    lower.includes('https://') ||
    lower.includes('github') ||
    lower.includes('checksum') ||
    lower.includes('sha256') ||
    lower.includes('sql') ||
    lower.includes('panic') ||
    lower.includes('nil pointer') ||
    lower.includes('stack trace') ||
    lower.includes('rpc error') ||
    lower.includes('i/o timeout') ||
    lower.includes('connection refused') ||
    lower.includes('.go') ||
    lower.includes('metaltracker/')
  );
}

/** True when the user dismissed a native file/save dialog (not a real failure). */
export function isUserCancelled(error: unknown): boolean {
  const text = normalizeErrorText(error);
  const parsed = parseAppError(text);
  if (parsed?.code === 'cancelled') {
    return true;
  }
  if (parsed && looksLikeUserCancelled(parsed.message)) {
    return true;
  }
  return looksLikeUserCancelled(stripErrorPrefix(text));
}

/** Capitalize the first letter and ensure a trailing period for short UI messages. */
export function formatUserMessage(message: string): string {
  let text = message.trim().replace(/\s+/g, ' ');
  if (!text) {
    return '';
  }
  text = text.charAt(0).toUpperCase() + text.slice(1);
  if (!/[.!?]$/.test(text)) {
    text += '.';
  }
  return text;
}

/**
 * Maps backend/Wails errors to short UI copy.
 * Never surfaces raw Go/HTTP/API dumps - use stable messages instead.
 */
export function formatAppError(error: unknown): string {
  if (isUserCancelled(error)) {
    return '';
  }

  const text = normalizeErrorText(error);
  const parsed = parseAppError(text);
  if (!parsed) {
    return codeMessage('internal');
  }

  if (parsed.code === 'cancelled') {
    return '';
  }

  // Validation is the only code that intentionally carries user-facing detail.
  if (parsed.code === 'validation') {
    const message = formatUserMessage(parsed.message);
    if (message && !looksTechnical(message)) {
      return message;
    }
    return codeMessage('validation');
  }

  return codeMessage(parsed.code);
}

export function appErrorCode(error: unknown): string {
  const parsed = parseAppError(normalizeErrorText(error));
  return parsed?.code ?? '';
}
