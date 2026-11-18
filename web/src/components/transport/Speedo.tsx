import { t } from '../../i18n/he';
import { gpsAt } from '../../lib/gps';
import { usePlayhead } from '../../hooks/useEngine';
import { useApp } from '../../store/app';

export function Speedo() {
  const track = useApp((s) => s.current?.detail?.track);
  const now = usePlayhead(8);
  const g = gpsAt(track, now);
  return (
    <div className="speedo" title={t.gpsSpeed}>
      <b>{g ? Math.round(g.kmh) : '--'}</b>
      <span>{t.kmh}</span>
    </div>
  );
}
