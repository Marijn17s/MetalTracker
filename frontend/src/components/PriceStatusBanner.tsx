import {useLocale} from '../i18n/LocaleContext';
import {formatDateTime} from '../utils/format';

interface PriceStatusBannerProps {
  quoteAsOf?: string;
  quoteIsStale?: boolean;
  quoteIsPartial?: boolean;
  valuationApproximate?: boolean;
  priceErrorCode?: string;
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
        {t('price.asOf', {when: formatDateTime(quoteAsOf)})}
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
    parts.push(t('price.lastQuote', {when: formatDateTime(quoteAsOf)}));
  }

  return (
    <div className="price-status price-status-warn" role="status">
      {parts.join(' ')}
    </div>
  );
}
