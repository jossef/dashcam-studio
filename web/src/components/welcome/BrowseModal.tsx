import { useEffect, useState } from 'react';
import { t } from '../../i18n/he';
import { api } from '../../lib/api';
import { useApp } from '../../store/app';
import { Modal } from '../common/Modal';
import { Icon } from '../Icon';

type Listing = Awaited<ReturnType<typeof api.browse>>;

/** Tiny server-side folder browser — browsers never reveal real paths from a native picker. */
export function BrowseModal({ start }: { start: string }) {
  const [res, setRes] = useState<Listing>({ path: '', parent: null, dirs: [] });

  const go = (p: string) =>
    api
      .browse(p)
      .catch((e: Error) => ({ path: p, parent: null, dirs: [], error: e.message }))
      .then(setRes);

  useEffect(() => {
    go(start);
  }, [start]);

  const pick = () => {
    useApp.getState().openModal(null);
    window.dispatchEvent(new CustomEvent('dm:pick-folder', { detail: res.path }));
  };

  return (
    <Modal
      title={t.browseTitle}
      footer={
        <>
          <span className="muted">{res.error ? res.error : res.isDashcam ? t.dashcamHere : ''}</span>
          <button className="btn primary" disabled={!res.path} onClick={pick}>
            {t.useFolder}
          </button>
        </>
      }
    >
      <div className="browse-path">
        <button className="icon-btn" title={t.up} disabled={!res.path} onClick={() => go(res.parent ?? '')}>
          <Icon name="up" />
        </button>
        <code dir="ltr">{res.path || t.thisComputer}</code>
      </div>
      <ul className="browse-list">
        {res.dirs.map((d) => (
          <li key={d.path}>
            <button className={/^(front|rear|cabin|normal|event|parking)$/i.test(d.name) ? 'cam' : ''} onClick={() => go(d.path)}>
              <Icon name={res.path ? 'folder' : 'drive'} />
              <span dir="ltr">{d.name}</span>
            </button>
          </li>
        ))}
      </ul>
    </Modal>
  );
}
