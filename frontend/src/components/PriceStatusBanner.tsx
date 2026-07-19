import {useLocale} from '../i18n/LocaleContext';

interface PriceStatusBannerProps {
  quoteAsOf?: string;
  quoteIsStale?: boolean;
  quoteIsPartial?: boolean;
  valuationApproximate?: boolean;
  priceErrorCode?: string;
}

function formatAsOf(value?: string): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

export function PriceStatusBanner({
  quoteAsOf,
  quoteIsStale,
  quoteIsPartial,
  valuationApproximate,
  priceErrorCode,
}: PriceStatusBannerProps) {
  const {t} = useLocale();

  if (!quoteIsStale && !quoteIsPartial && !valuationApproximate && !priceErrorCode) {
    if (!quoteAsOf) {
      return null;
    }
    return (
      <p className="price-status price-status-ok">
        {t('price.asOf', {when: formatAsOf(quoteAsOf)})}
      </p>
    );
  }

  const parts: string[] = [];
  if (priceErrorCode === 'invalid_api_key') {
    parts.push(t('price.needApiKey'));
  } else if (priceErrorCode === 'price_unavailable' || valuationApproximate) {
    parts.push(t('price.liveUnavailable'));
  }
  if (quoteIsStale) {
    parts.push(t('price.stale'));
  }
  if (quoteIsPartial) {
    parts.push(t('price.partial'));
  }
  if (quoteAsOf) {
    parts.push(t('price.lastQuote', {when: formatAsOf(quoteAsOf)}));
  }

  return (
    <div className="price-status price-status-warn" role="status">
      {parts.join(' ')}
    </div>
  );
}
