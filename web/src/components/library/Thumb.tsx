import { useEffect, useRef, useState, type ReactNode } from 'react';
import { urls } from '../../lib/api';
import { useApp } from '../../store/app';
import type { Clip } from '../../types';

let observer: IntersectionObserver | null = null;
const callbacks = new WeakMap<Element, () => void>();

/** One shared observer lazy-loads thumbnails as rows scroll into view. */
function observe(el: Element, cb: () => void) {
  observer ??= new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (!e.isIntersecting) continue;
        observer!.unobserve(e.target);
        callbacks.get(e.target)?.();
      }
    },
    { rootMargin: '300px' },
  );
  callbacks.set(el, cb);
  observer.observe(el);
  return () => observer?.unobserve(el);
}

export function Thumb({ clip, children }: { clip: Clip; children?: ReactNode }) {
  const ffmpeg = useApp((s) => !!s.status?.ffmpeg);
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);
  const [state, setState] = useState<'loading' | 'ok' | 'error'>('loading');
  const cam = clip.cams.F ? 'F' : Object.keys(clip.cams)[0];

  useEffect(() => (ffmpeg && ref.current ? observe(ref.current, () => setVisible(true)) : undefined), [ffmpeg]);

  return (
    <div ref={ref} className={`thumb ${state !== 'loading' || !ffmpeg ? 'loaded' : ''}`}>
      {visible && (
        <img alt="" decoding="async" className={state === 'ok' ? 'ok' : ''} src={urls.thumb(clip.id, cam)} onLoad={() => setState('ok')} onError={() => setState('error')} />
      )}
      {children}
    </div>
  );
}
