import { useEffect, useState, useSyncExternalStore } from 'react';
import { engine, type EngineSnapshot } from '../player/engine';

/** Re-renders when engine state (play/pause, rate, camera status…) changes. */
export function useEngine<T>(select: (s: EngineSnapshot) => T): T {
  return useSyncExternalStore(engine.subscribe, () => select(engine.getSnapshot()));
}

/**
 * Current playhead in seconds, sampled ~15×/s. Use only in small leaf
 * components (clock, timeline, telemetry) to keep re-renders cheap.
 */
export function usePlayhead(hz = 15): number {
  const [t, setT] = useState(0);
  useEffect(() => {
    let raf = 0;
    let last = 0;
    const loop = (ts: number) => {
      raf = requestAnimationFrame(loop);
      if (ts - last < 1000 / hz) return;
      last = ts;
      const now = engine.now();
      setT((prev) => (Math.abs(prev - now) > 0.001 ? now : prev));
    };
    raf = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(raf);
  }, [hz]);
  return t;
}
