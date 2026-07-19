import {ReactNode} from 'react';
import {useLocale} from '../i18n/LocaleContext';
import {TranslationKey} from '../i18n/translations';
import {AppView} from '../types';

interface ShellProps {
  view: AppView;
  holdingsStackActive?: boolean;
  soldStackActive?: boolean;
  onNavigate: (view: AppView) => void;
  onLock: () => void;
  children: ReactNode;
}

const navItems: {id: AppView; labelKey: TranslationKey; icon: ReactNode}[] = [
  {
    id: 'dashboard',
    labelKey: 'nav.dashboard',
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 13h7V4H4v9zm0 7h7v-5H4v5zm9 0h7V11h-7v9zm0-16v5h7V4h-7z" />
      </svg>
    ),
  },
  {
    id: 'holdings',
    labelKey: 'nav.holdings',
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 7h16v2H4V7zm0 4h16v2H4v-2zm0 4h10v2H4v-2z" />
      </svg>
    ),
  },
  {
    id: 'sold',
    labelKey: 'nav.sold',
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 4h16v2H4V4zm0 4h10v2H4V8zm0 4h16v2H4v-2zm0 4h10v2H4v-2zm12.5-1.5 3 3 5-5-1.4-1.4-3.6 3.6-1.6-1.6-1.4 1.4z" />
      </svg>
    ),
  },
  {
    id: 'add',
    labelKey: 'nav.add',
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M11 5h2v6h6v2h-6v6h-2v-6H5v-2h6V5z" />
      </svg>
    ),
  },
  {
    id: 'monthly',
    labelKey: 'nav.monthly',
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M6 4h2v2h8V4h2v2h2v14H4V6h2V4zm0 6v8h12v-8H6zm2 2h2v4H8v-4zm4 0h2v4h-2v-4z" />
      </svg>
    ),
  },
  {
    id: 'settings',
    labelKey: 'nav.settings',
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M19.14 12.94c.04-.31.06-.63.06-.94s-.02-.63-.06-.94l2.03-1.58a.5.5 0 0 0 .12-.64l-1.92-3.32a.5.5 0 0 0-.6-.22l-2.39.96a7.03 7.03 0 0 0-1.63-.94l-.36-2.54A.5.5 0 0 0 13.9 2h-3.8a.5.5 0 0 0-.5.42l-.36 2.54c-.58.23-1.12.54-1.63.94l-2.39-.96a.5.5 0 0 0-.6.22L2.7 8.48a.5.5 0 0 0 .12.64l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94L2.82 14.6a.5.5 0 0 0-.12.64l1.92 3.32c.13.23.4.32.6.22l2.39-.96c.5.4 1.05.72 1.63.94l.36 2.54c.05.24.26.42.5.42h3.8c.24 0 .45-.18.5-.42l.36-2.54c.58-.22 1.12-.54 1.63-.94l2.39.96c.23.1.47 0 .6-.22l1.92-3.32a.5.5 0 0 0-.12-.64l-2.03-1.58zM12 15.5A3.5 3.5 0 1 1 12 8a3.5 3.5 0 0 1 0 7.5z" />
      </svg>
    ),
  },
  {
    id: 'help',
    labelKey: 'nav.help',
    icon: (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm1 15h-2v-2h2v2zm1.07-7.75-.9.92A1.99 1.99 0 0 0 12 12h-1v-1c0-.55.22-1.05.59-1.41l1.24-1.26c.37-.36.58-.86.58-1.41 0-1.1-.9-2-2-2s-2 .9-2 2H8c0-2.21 1.79-4 4-4s4 1.79 4 4c0 .88-.36 1.68-.93 2.25z" />
      </svg>
    ),
  },
];

export function Shell({
  view,
  holdingsStackActive = false,
  soldStackActive = false,
  onNavigate,
  onLock,
  children,
}: ShellProps) {
  const {t} = useLocale();

  return (
    <div className="app-shell">
      <aside className="glass sidebar slide-in">
        <div className="sidebar-brand">
          <span className="brand-mark">{t('lock.brand')}</span>
          <p className="muted small">{t('shell.privateVault')}</p>
        </div>
        <nav className="sidebar-nav">
          {navItems.map((item) => {
            const active =
              view === item.id ||
              (item.id === 'holdings' && holdingsStackActive) ||
              (item.id === 'sold' && soldStackActive);
            return (
              <button
                key={item.id}
                type="button"
                className={`nav-item ${active ? 'active' : ''}`}
                onClick={() => onNavigate(item.id)}
              >
                <span className="nav-icon">{item.icon}</span>
                <span>{t(item.labelKey)}</span>
              </button>
            );
          })}
        </nav>
        <button type="button" className="btn btn-ghost lock-btn" onClick={onLock}>
          <span className="nav-icon">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M17 9V7a5 5 0 0 0-10 0v2H5v12h14V9h-2zm-8 0V7a3 3 0 0 1 6 0v2H9z" />
            </svg>
          </span>
          {t('nav.lock')}
        </button>
      </aside>
      <main className="main-content fade-in">{children}</main>
    </div>
  );
}
