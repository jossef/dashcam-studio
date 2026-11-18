import { fmtClock, fmtDur, fullDate } from '../../lib/format';
import { usePlayhead } from '../../hooks/useEngine';
import { engine } from '../../player/engine';
import { useApp } from '../../store/app';

/** Wall-clock time of the frame on screen + position in the clip. */
export function Clock() {
  const clip = useApp((s) => (s.current ? s.byId.get(s.current.id) : undefined));
  const now = usePlayhead(10);
  const wall = clip?.date ? new Date(clip.date.getTime() + now * 1000) : null;
  return (
    <div className="clock">
      <div className="wall" dir="ltr">
        {wall ? fmtClock(wall) : '--:--:--'}
      </div>
      <div className="pos">
        <span dir="ltr">
          {fmtDur(now)} / {fmtDur(engine.duration())}
        </span>
        {wall && <span className="date"> · {fullDate(wall)}</span>}
      </div>
    </div>
  );
}
