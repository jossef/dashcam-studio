import { memo } from 'react';
import { clamp } from '../../lib/format';
import type { TrackPoint } from '../../types';

const W = 1000;
const H = 100;

export const SpeedGraph = memo(function SpeedGraph({ track, duration }: { track?: TrackPoint[]; duration: number }) {
  if (!track?.length) return <svg className="speed-graph" />;
  const maxV = Math.max(50, ...track.map((p) => p[3])) * 1.12;
  const pts = track.filter((p) => p[0] >= 0 && p[0] <= duration + 1).map((p) => [clamp(p[0] / duration, 0, 1) * W, H - (p[3] / maxV) * H]);
  if (!pts.length) return <svg className="speed-graph" />;
  const line = pts.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join('');
  const last = pts[pts.length - 1];
  return (
    <svg className="speed-graph" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden="true">
      <defs>
        <linearGradient id="spd-grad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#ffb238" stopOpacity=".32" />
          <stop offset="1" stopColor="#ffb238" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path className="area" d={`${line}L${last[0].toFixed(1)},${H}L${pts[0][0].toFixed(1)},${H}Z`} />
      <path className="line" d={line} />
    </svg>
  );
});
