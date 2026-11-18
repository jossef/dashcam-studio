import type { TrackPoint } from '../types';

export interface GpsSample {
  lat: number;
  lon: number;
  kmh: number;
  hdg: number;
}

/** Interpolated position at `t` seconds into the clip. */
export function gpsAt(track: TrackPoint[] | undefined, t: number): GpsSample | null {
  if (!track?.length) return null;
  const pt = (p: TrackPoint) => ({ lat: p[1], lon: p[2], kmh: p[3], hdg: p[4] });
  if (t <= track[0][0]) return pt(track[0]);
  let lo = 0;
  let hi = track.length - 1;
  if (t >= track[hi][0]) return pt(track[hi]);
  while (hi - lo > 1) {
    const mid = (lo + hi) >> 1;
    if (track[mid][0] <= t) lo = mid;
    else hi = mid;
  }
  const a = track[lo];
  const b = track[hi];
  const k = (t - a[0]) / Math.max(0.001, b[0] - a[0]);
  let hdg = b[4];
  if (a[3] > 3 || b[3] > 3) {
    // bearing from movement is steadier than the logged course
    const r = Math.PI / 180;
    const y = Math.sin((b[2] - a[2]) * r) * Math.cos(b[1] * r);
    const x = Math.cos(a[1] * r) * Math.sin(b[1] * r) - Math.sin(a[1] * r) * Math.cos(b[1] * r) * Math.cos((b[2] - a[2]) * r);
    if (x || y) hdg = (Math.atan2(y, x) / r + 360) % 360;
  }
  return { lat: a[1] + (b[1] - a[1]) * k, lon: a[2] + (b[2] - a[2]) * k, kmh: a[3] + (b[3] - a[3]) * k, hdg };
}

export function nearestIndex(track: TrackPoint[], lat: number, lon: number): number {
  let best = 0;
  let bd = Infinity;
  track.forEach((p, i) => {
    const d = (p[1] - lat) ** 2 + (p[2] - lon) ** 2;
    if (d < bd) {
      bd = d;
      best = i;
    }
  });
  return best;
}
