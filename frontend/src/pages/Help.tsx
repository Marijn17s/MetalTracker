import {useLocale} from '../i18n/LocaleContext';

export function Help() {
  const {t} = useLocale();

  return (
    <div className="page-stack">
      <header className="page-header">
        <div>
          <p className="eyebrow">{t('help.eyebrow')}</p>
          <h1>{t('help.title')}</h1>
        </div>
      </header>

      <div className="help-sections">
        <article className="glass panel">
          <h2>{t('help.vaultTitle')}</h2>
          <p>{t('help.vaultBody')}</p>
        </article>
        <article className="glass panel">
          <h2>{t('help.pinTitle')}</h2>
          <p>{t('help.pinBody')}</p>
        </article>
        <article className="glass panel">
          <h2>{t('help.recoveryTitle')}</h2>
          <p>{t('help.recoveryBody')}</p>
        </article>
        <article className="glass panel">
          <h2>{t('help.pricesTitle')}</h2>
          <p>{t('help.pricesBody')}</p>
        </article>
        <article className="glass panel">
          <h2>{t('help.backupTitle')}</h2>
          <p>{t('help.backupBody')}</p>
        </article>
        <article className="glass panel">
          <h2>{t('help.migrateTitle')}</h2>
          <p>{t('help.migrateBody')}</p>
        </article>
        <article className="glass panel">
          <h2>{t('help.updatesTitle')}</h2>
          <p>{t('help.updatesBody')}</p>
        </article>
      </div>
    </div>
  );
}
