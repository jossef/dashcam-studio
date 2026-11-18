import { memo, useEffect, useRef } from 'react';
import { camName, categoryName, t } from '../../i18n/he';
import { fmtClock, fmtKm } from '../../lib/format';
import { useEngine } from '../../hooks/useEngine';
import { openClip } from '../../store/actions';
import { useApp } from '../../store/app';
import { CAMS, type Clip } from '../../types';
import { Checkbox } from '../common/Checkbox';
import { StarButton } from '../common/StarButton';
import { Thumb } from './Thumb';

export const ClipRow = memo(function ClipRow({ clip, order, showCategory = false }: { clip: Clip; order: string[]; showCategory?: boolean }) {
  const active = useApp((s) => s.current?.id === clip.id);
  const selected = useApp((s) => s.selection.has(clip.id));
  const selecting = useApp((s) => s.selection.size > 0);
  const playing = useEngine((s) => s.playing);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!active || !ref.current) return;
    const row = ref.current.getBoundingClientRect();
    const list = ref.current.closest('.lib-list')?.getBoundingClientRect();
    if (list && (row.top < list.top + 40 || row.bottom > list.bottom)) ref.current.scrollIntoView({ block: 'center', behavior: 'smooth' });
  }, [active]);

  const toggle = (shift: boolean) => useApp.getState().toggleSelect(clip.id, shift ? { range: order } : undefined);

  return (
    <div
      ref={ref}
      role="button"
      tabIndex={0}
      data-id={clip.id}
      className={`clip ${active ? 'active' : ''} ${active && playing ? 'playing' : ''} ${selected ? 'selected' : ''} ${selecting ? 'selecting' : ''}`}
      onClick={(e) => {
        if (e.ctrlKey || e.metaKey || e.shiftKey || selecting) toggle(e.shiftKey);
        else openClip(clip.id, { play: true });
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter') openClip(clip.id, { play: true });
      }}
    >
      <Thumb clip={clip}>
        <Checkbox state={selected} label={t.selectClip} onToggle={(e) => toggle(e.shiftKey)} />
      </Thumb>
      <div className="clip-info">
        <div className="clip-time">
          <span>{clip.date ? fmtClock(clip.date) : clip.stem}</span>
          <span className="eq" aria-hidden="true">
            <i />
            <i />
            <i />
          </span>
        </div>
        <div className="clip-sub">
          <span className="cams">
            {CAMS.map((k) => (
              <i key={k} className={`cam-${k} ${clip.cams[k] ? '' : 'off'}`} title={clip.cams[k] ? camName[k] : t.missingCam(k)} />
            ))}
          </span>
          {showCategory && <span className="cat">{categoryName(clip.category)}</span>}
          {clip.gps && (
            <>
              <span>
                {fmtKm(clip.gps.km)} {t.km}
              </span>
              <span>▲{Math.round(clip.gps.maxKmh)}</span>
            </>
          )}
        </div>
      </div>
      <StarButton id={clip.id} />
    </div>
  );
});
