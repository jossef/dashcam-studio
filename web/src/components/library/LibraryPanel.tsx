import { useMemo, useRef } from 'react';
import { t } from '../../i18n/he';
import { groupByDay, newestFirst } from '../../lib/library';
import { isFavoritesView, useApp } from '../../store/app';
import type { Clip, DayGroup, Trip } from '../../types';
import { DayHeader } from './DayHeader';
import { DayStrip } from './DayStrip';
import { FavoritesHeader } from './FavoritesHeader';
import { ClipRow } from './ClipRow';
import { SelectionBar } from './SelectionBar';
import { TripGroup } from './TripGroup';

/**
 * The clip list. Everything is ordered newest → oldest (top → bottom):
 * days, the trips inside a day, and the clips inside a trip.
 */
export function LibraryPanel() {
  const view = useApp((s) => s.view);
  const trips = useApp((s) => s.trips);
  const clips = useApp((s) => s.clips);
  const favorites = useApp((s) => s.favorites);
  const listRef = useRef<HTMLDivElement>(null);
  const fav = isFavoritesView(view);

  const tripDays: DayGroup<Trip>[] = useMemo(
    () => (fav || !view ? [] : groupByDay(trips.get(view) ?? [], (tr) => tr.start)),
    [fav, view, trips],
  );
  const favClips = useMemo(() => (fav ? newestFirst(clips.filter((c) => favorites.has(c.id))) : []), [fav, clips, favorites]);
  const favDays: DayGroup<Clip>[] = useMemo(() => groupByDay(favClips, (c) => c.date), [favClips]);

  /** visible order, for shift-click range selection */
  const order = useMemo(
    () => (fav ? favClips.map((c) => c.id) : tripDays.flatMap((d) => d.items.flatMap((tr) => [...tr.clips].reverse().map((c) => c.id)))),
    [fav, favClips, tripDays],
  );

  const days = fav ? favDays : tripDays;
  const empty = fav ? !favClips.length : !tripDays.length;

  return (
    <aside className="library" aria-label="קטעים">
      <div className="lib-head">
        <DayStrip days={days.map((d) => ({ day: d.day, date: d.date }))} listRef={listRef} />
      </div>
      <div className="lib-list" ref={listRef} tabIndex={-1}>
        {fav && <FavoritesHeader clips={favClips} />}
        {empty &&
          (fav ? (
            <div className="lib-empty">
              <b>{t.favEmptyTitle}</b>
              <span>{t.favEmptyBody}</span>
            </div>
          ) : (
            <div className="lib-empty">{t.noRecordings}</div>
          ))}
        {fav
          ? favDays.map((g) => (
              <section key={g.day} data-day={g.day}>
                <DayHeader date={g.date} ids={g.items.map((c) => c.id)} km={g.items.reduce((s, c) => s + (c.gps?.km || 0), 0)} />
                <div className="fav-list">
                  {g.items.map((c) => (
                    <ClipRow key={c.id} clip={c} order={order} showCategory />
                  ))}
                </div>
              </section>
            ))
          : tripDays.map((g) => (
              <section key={g.day} data-day={g.day}>
                <DayHeader
                  date={g.date}
                  ids={g.items.flatMap((tr) => tr.clips.map((c) => c.id))}
                  km={g.items.reduce((s, tr) => s + tr.km, 0)}
                  trips={g.items.length}
                />
                {g.items.map((tr) => (
                  <TripGroup key={tr.id} trip={tr} order={order} />
                ))}
              </section>
            ))}
      </div>
      <SelectionBar />
    </aside>
  );
}
