/** Max-speed bars per clip, drawn left→right in time (like the timeline). */
export function Sparkline({ values, max }: { values: number[]; max: number }) {
  const w = 100;
  const h = 18;
  const bw = w / values.length;
  return (
    <svg className="trip-spark" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" aria-hidden="true">
      {values.map((v, i) => (
        <rect key={i} x={i * bw + bw * 0.15} y={h - (v / max) * h} width={Math.max(0.6, bw * 0.7)} height={(v / max) * h} rx={0.4} />
      ))}
    </svg>
  );
}
