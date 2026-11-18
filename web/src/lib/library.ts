import type { Clip, DayGroup, Trip } from '../types';
import { clipBytes, dayKey, sum } from './format';

/** Clips that start within this many seconds of the previous one belong to the same drive. */
export const TRIP_GAP_S = 95;

const time = (c: Clip) => (c.date ? c.date.getTime() : null);

/** Chronological (oldest first) — the playback order. Clip ids break ties. */
export function chronological(clips: Clip[]): Clip[] {
  return [...clips].sort((a, b) => (time(a) ?? 0) - (time(b) ?? 0) || a.seq - b.seq || a.id.localeCompare(b.id));
}

/** Group a category's clips into drives. Returns trips NEWEST FIRST (the list order). */
export function buildTrips(clips: Clip[]): Trip[] {
  const trips: Trip[] = [];
  let cur: Trip | null = null;
  let lastT: number | null = null;
  for (const c of chronological(clips)) {
    const t = time(c);
    const gap = t == null || lastT == null ? Infinity : (t - lastT) / 1000;
    if (!cur || cur.category !== c.category || gap > TRIP_GAP_S || gap < 0) {
      cur = { id: c.id, category: c.category, clips: [], start: null, end: null, km: 0, maxKmh: 0, bytes: 0, day: '' };
      trips.push(cur);
    }
    cur.clips.push(c);
    lastT = t;
  }
  for (const tr of trips) {
    tr.start = tr.clips[0].date;
    const last = tr.clips[tr.clips.length - 1].date;
    tr.end = last ? new Date(last.getTime() + 60_000) : null;
    tr.km = sum(tr.clips, (c) => c.gps?.km);
    tr.maxKmh = Math.max(0, ...tr.clips.map((c) => c.gps?.maxKmh || 0));
    tr.bytes = sum(tr.clips, clipBytes);
    tr.day = tr.start ? dayKey(tr.start) : 'unknown';
  }
  return trips.reverse();
}

/** Group already-sorted items by calendar day, keeping their order. */
export function groupByDay<T>(items: T[], dateOf: (x: T) => Date | null): DayGroup<T>[] {
  const out: DayGroup<T>[] = [];
  const index = new Map<string, DayGroup<T>>();
  for (const it of items) {
    const d = dateOf(it);
    const key = d ? dayKey(d) : 'unknown';
    let g = index.get(key);
    if (!g) {
      g = { day: key, date: d, items: [] };
      index.set(key, g);
      out.push(g);
    }
    g.items.push(it);
  }
  return out;
}

/** Newest first — the order every list in the UI uses. */
export const newestFirst = (clips: Clip[]) => chronological(clips).reverse();
