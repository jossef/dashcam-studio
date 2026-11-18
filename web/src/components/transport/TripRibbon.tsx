import { t } from '../../i18n/he';
import { fmtClock, fmtHM, fmtKm } from '../../lib/format';
import { openClip } from '../../store/actions';
import { useApp } from '../../store/app';

/** The current drive as a strip of clips — chronological, left → right like the timeline. */
export function TripRibbon() {
  const id = useApp((s) => s.current?.id);
  const trip = useApp((s) => (s.current ? s.tripOf.get(s.current.id) : undefined));
  const favorites = useApp((s) => s.favorites);
  if (!trip || !id) return <div className="trip-ribbon" />;
  const idx = trip.clips.findIndex((c) => c.id === id);
  return (
    <div className="trip-ribbon">
      <div className="trip-label">
        {t.trip}{' '}
        <b dir="ltr">
          {trip.start ? fmtHM(trip.start) : ''}–{trip.end ? fmtHM(trip.end) : ''}
        </b>{' '}
        · {idx + 1}/{trip.clips.length}
        {trip.km > 0 && ` · ${fmtKm(trip.km)} ${t.km}`}
      </div>
      <div className="trip-cells" dir="ltr">
        {trip.clips.map((c, i) => (
          <button
            key={c.id}
            className={`${i < idx ? 'done' : i === idx ? 'cur' : ''} ${c.cams.C ? '' : 'nocabin'} ${favorites.has(c.id) ? 'fav' : ''}`}
            title={`${c.date ? fmtClock(c.date) : c.stem}${c.gps ? ` · ${fmtKm(c.gps.km)} ${t.km} · ▲${Math.round(c.gps.maxKmh)} ${t.kmh}` : ''}`}
            onClick={() => openClip(c.id, { play: true })}
          />
        ))}
      </div>
    </div>
  );
}
