import { clipsN, t, tripsN } from '../../i18n/he';
import { dayTitle, fmtKm } from '../../lib/format';
import { useApp } from '../../store/app';
import { Checkbox, selectionState } from '../common/Checkbox';

export function DayHeader({ date, ids, km, trips }: { date: Date | null; ids: string[]; km: number; trips?: number }) {
  const state = useApp((s) => selectionState(ids, s.selection));
  const anySelected = useApp((s) => s.selection.size > 0);
  return (
    <div className={`lib-day ${anySelected ? 'selecting' : ''}`}>
      <Checkbox state={state} label={t.selectDay} onToggle={() => useApp.getState().setSelected(ids, state !== true)} />
      <span className="lib-day-title">{date ? dayTitle(date) : t.unknownDate}</span>
      <small>
        {trips != null && `${tripsN(trips)} · `}
        {clipsN(ids.length)}
        {km > 0 && ` · ${fmtKm(km)} ${t.km}`}
      </small>
    </div>
  );
}
