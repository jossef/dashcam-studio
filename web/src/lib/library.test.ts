import { describe, expect, it } from 'vitest';
import type { Clip } from '../types';
import { buildTrips, groupByDay, newestFirst } from './library';

const clip = (stem: string, iso: string, category = 'Normal'): Clip => ({
  id: `${category}/${stem}`, stem, category, start: iso, seq: Number(stem.slice(-3)), cams: { F: 10, R: 5 }, date: new Date(iso),
});

const clips = [
  clip('A001', '2026-11-09T19:14:23'),
  clip('A002', '2026-11-09T19:15:23'),
  clip('A003', '2026-11-09T19:16:23'),
  clip('B004', '2026-11-09T21:00:00'),
  clip('C005', '2026-11-13T08:00:00'),
];

describe('buildTrips', () => {
  it('groups consecutive clips and lists the newest trip first', () => {
    const trips = buildTrips([...clips].reverse());
    expect(trips.map((t) => t.clips.map((c) => c.stem))).toEqual([['C005'], ['B004'], ['A001', 'A002', 'A003']]);
    expect(trips[2].end?.toISOString()).toBe(new Date('2026-11-09T19:17:23').toISOString());
    expect(trips[2].bytes).toBe(45);
  });
});

describe('newestFirst + groupByDay', () => {
  it('orders latest at the top, oldest at the bottom', () => {
    const groups = groupByDay(newestFirst(clips), (c) => c.date);
    expect(groups.map((g) => g.day)).toEqual(['2026-11-13', '2026-11-09']);
    expect(groups[1].items.map((c) => c.stem)).toEqual(['B004', 'A003', 'A002', 'A001']);
  });
});
