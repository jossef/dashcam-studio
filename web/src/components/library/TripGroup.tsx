import { memo, useEffect, useState } from 'react';
import { clipsN, t } from '../../i18n/he';
import { fmtHM, fmtKm } from '../../lib/format';
import { useApp } from '../../store/app';
import type { Trip } from '../../types';
import { Checkbox, selectionState } from '../common/Checkbox';
import { Icon } from '../Icon';
import { ClipRow } from './ClipRow';
import { Sparkline } from './Sparkline';

export const TripGroup = memo(function TripGroup({ trip, order }: { trip: Trip; order: string[] }) {
  const ids = trip.clips.map((c) => c.id);
  const isCurrent = useApp((s) => !!s.current && s.tripOf.get(s.current.id) === trip);
  const sel = useApp((s) => selectionState(ids, s.selection));
  const anySelected = useApp((s) => s.selection.size > 0);
  const starred = useApp((s) => ids.filter((id) => s.favorites.has(id)).length);
  const [open, setOpen] = useState(isCurrent);

  useEffect(() => {
    if (isCurrent) setOpen(true);
  }, [isCurrent]);

  // newest clip at the top
  const rows = open ? [...trip.clips].reverse() : [];

  return (
    <section className={`trip ${open ? 'open' : ''} ${isCurrent ? 'current' : ''}`}>
      <div className={`trip-head ${anySelected ? 'selecting' : ''}`} role="button" tabIndex={0} aria-expanded={open} onClick={() => setOpen(!open)} onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), setOpen(!open))}>
        <Checkbox state={sel} label={t.selectTrip} onToggle={() => useApp.getState().setSelected(ids, sel !== true)} />
        <Icon name="chev" className="icon chev" />
        <span className="trip-title">
          {trip.start ? fmtHM(trip.start) : '?'}
          <span className="arrow">←</span>
          {trip.end ? fmtHM(trip.end) : ''}
        </span>
        <span className="trip-meta">
          {starred > 0 && (
            <span className="trip-stars" title={t.favorites}>
              ★{starred}
            </span>
          )}
          {clipsN(trip.clips.length)}
          {trip.km > 0 && ` · ${fmtKm(trip.km)} ${t.km}`}
        </span>
        {trip.maxKmh > 0 && <Sparkline values={trip.clips.map((c) => c.gps?.maxKmh || 0)} max={trip.maxKmh} />}
      </div>
      {open && (
        <div className="trip-body">
          {rows.map((c) => (
            <ClipRow key={c.id} clip={c} order={order} />
          ))}
        </div>
      )}
    </section>
  );
});
