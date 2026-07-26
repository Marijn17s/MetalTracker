import {Environment} from '../../wailsjs/runtime/runtime';
import {UITheme} from '../types';

export function applyTheme(theme: UITheme | string | undefined) {
  const resolved = theme === 'light' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', resolved);
}

/** Linux webview runs without GPU, so backdrop-filter blur is ineffective. */
export async function applyPlatformClass(): Promise<void> {
  try {
    const environment = await Environment();
    document.documentElement.dataset.platform = environment.platform;
  } catch {
    // Running outside the Wails runtime (e.g. browser preview).
  }
}
