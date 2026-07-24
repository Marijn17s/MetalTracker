import {createContext, ReactNode, useContext, useMemo} from 'react';
import {INTL_LOCALE, translate, TranslationKey} from './translations';

interface LocaleContextValue {
  t: (key: TranslationKey, replacements?: Record<string, string | number>) => string;
  /** Undefined means formatting follows the OS/runtime locale. */
  intl: string | undefined;
}

const LocaleContext = createContext<LocaleContextValue | null>(null);

export function LocaleProvider({children}: {children: ReactNode}) {
  const value = useMemo<LocaleContextValue>(() => ({
    t: translate,
    intl: INTL_LOCALE,
  }), []);

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale(): LocaleContextValue {
  const context = useContext(LocaleContext);
  if (!context) {
    throw new Error('useLocale must be used within LocaleProvider');
  }
  return context;
}
