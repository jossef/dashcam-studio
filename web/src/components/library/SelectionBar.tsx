import { t } from '../../i18n/he';
import { fmtBytes } from '../../lib/format';
import { openExport, selectionBytes } from '../../store/actions';
import { useApp } from '../../store/app';
import { Icon } from '../Icon';

/** Appears at the bottom of the library while clips are selected. */
export function SelectionBar() {
  const selection = useApp((s) => s.selection);
  const allStarred = useApp((s) => [...s.selection].every((id) => s.favorites.has(id)));
  if (!selection.size) return null;
  const ids = [...selection];
  const { setFavorite, clearSelection } = useApp.getState();

  return (
    <div className="sel-bar" role="region" aria-label={t.selected(ids.length)}>
      <div className="sel-count">
        <b>{t.selected(ids.length)}</b>
        <small>{fmtBytes(selectionBytes(ids))}</small>
      </div>
      <button className="btn primary" onClick={() => openExport(ids)}>
        <Icon name="zip" />
        {t.exportSelected}
      </button>
      <button className="icon-btn" title={allStarred ? t.unstarSelected : t.starSelected} onClick={() => setFavorite(ids, !allStarred)}>
        <Icon name={allStarred ? 'starFill' : 'star'} />
      </button>
      <button className="icon-btn" title={t.clearSelection} onClick={clearSelection}>
        <Icon name="x" />
      </button>
    </div>
  );
}
