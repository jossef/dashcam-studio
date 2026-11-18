import { useEffect, useRef, useState } from 'react';
import { t } from '../../i18n/he';
import { clamp, fmtClock, fmtDur } from '../../lib/format';
import { gpsAt } from '../../lib/gps';
import { usePlayhead } from '../../hooks/useEngine';
import { engine } from '../../player/engine';
import { useApp } from '../../store/app';
import { SpeedGraph } from './SpeedGraph';

/** Seek bar with the clip's speed graph. Always left → right (media convention), even in RTL. */
export function Timeline() {
  const detail = useApp((s) => s.current?.detail);
  const clip = useApp((s) => (s.current ? s.byId.get(s.current.id) : undefined));
  const now = usePlayhead();
  const ref = useRef<HTMLDivElement>(null);
  const [hover, setHover] = useState<number | null>(null);
  const [drag, setDrag] = useState(false);
  const drag$ = useRef({ resume: false, last: 0 });
  const dur = engine.duration();
  const f = clamp(now / dur, 0, 1);
  const buf = clamp(engine.bufferedEnd(now) / dur, 0, 1);

  const frac = (clientX: number) => {
    const r = ref.current!.getBoundingClientRect();
    return clamp((clientX - r.left) / r.width, 0, 1);
  };

  useEffect(() => {
    const el = ref.current!;
    const onWheel = (e: WheelEvent) => {
      if (!useApp.getState().current?.detail) return;
      e.preventDefault();
      engine.seek(engine.now() + (e.deltaY > 0 ? 2 : -2) * (e.shiftKey ? 5 : 1));
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => el.removeEventListener('wheel', onWheel);
  }, []);

  const describe = (sec: number) => {
    const parts: string[] = [];
    if (clip?.date) parts.push(fmtClock(new Date(clip.date.getTime() + sec * 1000)));
    parts.push(fmtDur(sec));
    const g = gpsAt(detail?.track, sec);
    if (g) parts.push(`${Math.round(g.kmh)} ${t.kmh}`);
    return parts.join(' · ');
  };

  return (
    <div
      ref={ref}
      dir="ltr"
      className={`timeline ${drag ? 'drag' : ''}`}
      role="slider"
      aria-label={t.seek}
      aria-valuemin={0}
      aria-valuemax={Math.round(dur)}
      aria-valuenow={Math.round(now)}
      tabIndex={0}
      onPointerMove={(e) => {
        if (!detail) return;
        const x = frac(e.clientX);
        setHover(x);
        if (drag && performance.now() - drag$.current.last > 90) {
          drag$.current.last = performance.now();
          engine.seek(x * dur);
        }
      }}
      onPointerLeave={() => setHover(null)}
      onPointerDown={(e) => {
        if (!detail) return;
        setDrag(true);
        drag$.current.resume = engine.playing;
        ref.current!.setPointerCapture(e.pointerId);
        if (engine.playing) engine.pause();
        engine.seek(frac(e.clientX) * dur);
      }}
      onPointerUp={(e) => {
        if (!drag) return;
        setDrag(false);
        engine.seek(frac(e.clientX) * dur);
        if (drag$.current.resume) engine.play();
      }}
      onPointerCancel={() => setDrag(false)}
    >
      <SpeedGraph track={detail?.track} duration={dur} />
      <div className="tl-rail" />
      <div className="tl-buffer" style={{ width: `${buf * 100}%` }} />
      <div className="tl-progress" style={{ width: `${f * 100}%` }} />
      <div className="tl-head" style={{ left: `${f * 100}%` }} />
      {hover != null && (
        <div className="tl-hover" style={{ left: `${hover * 100}%` }}>
          <span>{describe(hover * dur)}</span>
        </div>
      )}
    </div>
  );
}
