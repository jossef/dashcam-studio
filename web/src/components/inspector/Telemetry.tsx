import { t } from '../../i18n/he';
import { compass, fmtKm } from '../../lib/format';
import { gpsAt } from '../../lib/gps';
import { usePlayhead } from '../../hooks/useEngine';
import { useApp } from '../../store/app';
import { Icon } from '../Icon';

export function Telemetry() {
  const track = useApp((s) => s.current?.detail?.track);
  const clip = useApp((s) => (s.current ? s.byId.get(s.current.id) : undefined));
  const now = usePlayhead(8);
  const g = gpsAt(track, now);
  const coords = g ? `${g.lat.toFixed(5)}, ${g.lon.toFixed(5)}` : '';

  return (
    <>
      <div className="telemetry">
        <div className="tele">
          <span className="eyebrow">{t.speedLbl}</span>
          <b>{g ? Math.round(g.kmh) : '--'}</b>
          <small>{t.kmh}</small>
        </div>
        <div className="tele">
          <span className="eyebrow">{t.heading}</span>
          <b className="heading">
            {g ? (
              <>
                <span className="needle" style={{ transform: `rotate(${g.hdg}deg)` }}>
                  ↑
                </span>
                {Math.round(g.hdg)}°
              </>
            ) : (
              '--'
            )}
          </b>
          <small>{g ? compass(g.hdg) : ''}</small>
        </div>
        <div className="tele">
          <span className="eyebrow">{t.clipDist}</span>
          <b>{clip?.gps ? fmtKm(clip.gps.km) : '--'}</b>
          <small>{t.km}</small>
        </div>
        <div className="tele">
          <span className="eyebrow">{t.topSpeed}</span>
          <b>{clip?.gps ? Math.round(clip.gps.maxKmh) : '--'}</b>
          <small>{t.kmh}</small>
        </div>
      </div>
      <div className="coords" style={{ opacity: g ? 1 : 0.5 }}>
        <Icon name="pin" />
        <span dir="ltr">{g ? coords : t.noFix}</span>
        <button
          className="mini"
          title={t.copyCoords}
          disabled={!g}
          onClick={() => navigator.clipboard?.writeText(coords).then(() => useApp.getState().toast(t.copied))}
        >
          <Icon name="copy" />
        </button>
        <a className="mini" target="_blank" rel="noopener" title={t.openMaps} href={g ? `https://www.google.com/maps?q=${g.lat.toFixed(6)},${g.lon.toFixed(6)}` : undefined}>
          {t.maps}
        </a>
      </div>
    </>
  );
}
