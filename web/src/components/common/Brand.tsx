import { t } from '../../i18n/he';

export function Brand({ large = false }: { large?: boolean }) {
  return (
    <div className={`brand ${large ? 'brand-xl' : ''}`} dir="ltr">
      <span className="brand-name">
        DASHCAM<b>STUDIO</b>
      </span>
      {large && <span className="brand-tag" dir="rtl">{t.tagline}</span>}
    </div>
  );
}
