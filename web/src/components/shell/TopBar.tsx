import { categoryName, t } from '../../i18n/he';
import { rescan } from '../../store/actions';
import { isFavoritesView, useApp } from '../../store/app';
import { FAVORITES_VIEW } from '../../types';
import { Brand } from '../common/Brand';
import { Icon } from '../Icon';

export function TopBar() {
  const root = useApp((s) => s.root);
  const categories = useApp((s) => s.categories);
  const view = useApp((s) => s.view);
  const favCount = useApp((s) => [...s.favorites].filter((id) => s.byId.has(id)).length);
  const showLibrary = useApp((s) => s.showLibrary);
  const showInspector = useApp((s) => s.showInspector);
  const removable = useApp((s) => !!s.status?.removable);
  const { setView, togglePanel, openModal, setWelcome } = useApp.getState();

  return (
    <header className="topbar">
      <Brand />
      <button className="root-chip" title={`${root} — ${t.openOther}`} onClick={() => setWelcome(true)}>
        <Icon name="drive" />
        <span dir="ltr">{root}</span>
      </button>
      <button className={`icon-btn eject-btn ${removable ? 'removable' : ''}`} title={t.disconnect} onClick={() => openModal({ kind: 'disconnect' })}>
        <Icon name="eject" />
      </button>
      <nav className="cat-tabs" aria-label={t.recordingType}>
        {categories.map((c) => (
          <button key={c.name} className={view === c.name ? 'active' : ''} disabled={!c.count} onClick={() => setView(c.name)}>
            {categoryName(c.name)}
            <small>{c.count}</small>
          </button>
        ))}
        <button className={`fav-tab ${isFavoritesView(view) ? 'active' : ''}`} onClick={() => setView(FAVORITES_VIEW)}>
          <Icon name={favCount ? 'starFill' : 'star'} />
          {t.favorites}
          <small>{favCount}</small>
        </button>
      </nav>
      <div className="spacer" />
      <div className="top-actions">
        <button className="icon-btn" title={t.rescan} onClick={rescan}>
          <Icon name="refresh" />
        </button>
        <button className={`icon-btn ${showLibrary ? 'on' : ''}`} title={t.toggleLibrary} onClick={() => togglePanel('showLibrary')}>
          <Icon name="list" />
        </button>
        <button className={`icon-btn ${showInspector ? 'on' : ''}`} title={t.toggleInspector} onClick={() => togglePanel('showInspector')}>
          <Icon name="map" />
        </button>
        <button className="icon-btn" title={t.shortcuts} onClick={() => openModal({ kind: 'help' })}>
          <Icon name="keys" />
        </button>
      </div>
    </header>
  );
}
