import { useEffect, useRef, type RefObject } from 'react';
import { monthShort, weekdayShort } from '../../lib/format';
import { useApp } from '../../store/app';

/** Horizontal strip of day chips (newest first, starting at the right in RTL). */
export function DayStrip({ days, listRef }: { days: { day: string; date: Date | null }[]; listRef: RefObject<HTMLDivElement | null> }) {
  const currentDay = useApp((s) => {
    const tr = s.current && s.tripOf.get(s.current.id);
    return tr?.day ?? null;
  });
  const strip = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = strip.current;
    if (!el) return;
    const onWheel = (e: WheelEvent) => {
      if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
        e.preventDefault();
        el.scrollLeft -= e.deltaY; // RTL: scrolling down moves towards older days (left)
      }
    };
    el.addEventListener('wheel', onWheel, { passive: false });
    return () => el.removeEventListener('wheel', onWheel);
  }, []);

  const jump = (day: string) =>
    listRef.current?.querySelector(`[data-day="${day}"]`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });

  return (
    <div className="day-strip" ref={strip}>
      {days.map(({ day, date }) => (
        <button key={day} className={`day-chip ${day === currentDay ? 'active' : ''}`} onClick={() => jump(day)}>
          {date ? (
            <>
              <span>{weekdayShort(date)}</span>
              <b>{date.getDate()}</b>
              <small>{monthShort(date)}</small>
            </>
          ) : (
            <b>?</b>
          )}
        </button>
      ))}
    </div>
  );
}
