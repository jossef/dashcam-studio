import { t } from '../../i18n/he';
import { clipBytes, fmtBytes, sum } from '../../lib/format';
import { openExport } from '../../store/actions';
import { useApp } from '../../store/app';
import type { Clip } from '../../types';
import { Icon } from '../Icon';

export function FavoritesHeader({ clips }: { clips: Clip[] }) {
  const missing = useApp((s) => [...s.favorites].filter((id) => !s.byId.has(id)).length);
  if (!clips.length && !missing) return null;
  return (
    <div className="fav-head">
      {clips.length > 0 && (
        <button className="btn primary fav-export" onClick={() => openExport(clips.map((c) => c.id))}>
          <Icon name="zip" />
          <span>{t.exportAllFavs}</span>
          <small>
            {clips.length} · {fmtBytes(sum(clips, clipBytes))}
          </small>
        </button>
      )}
      <p className="fav-warn">
        <Icon name="warn" />
        <span>
          {t.favLoopWarn}
          {missing > 0 && <em> {t.favMissing(missing)}.</em>}
        </span>
      </p>
    </div>
  );
}
