import {UITheme} from '../types';

export function applyTheme(theme: UITheme | string | undefined) {
  const resolved = theme === 'light' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', resolved);
}
