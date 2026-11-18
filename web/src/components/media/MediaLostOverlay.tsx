import { useState } from 'react';
import { t } from '../../i18n/he';
import { fmtClock } from '../../lib/format';
import { disconnectMedia } from '../../store/actions';
import { useApp } from '../../store/app';
import { Icon } from '../Icon';

/** Shown when the card disappears. The app keeps polling and resumes on its own when it returns. */
export function MediaLostOverlay() {
  const offline = useApp((s) => s.offline);
  const clip = useApp((s) => (s.offline?.resume ? s.byId.get(s.offline.resume.id) : undefined));
  const [busy, setBusy] = useState(false);
  if (!offline) return null;
  const at = clip?.date && offline.resume ? new Date(clip.date.getTime() + offline.resume.t * 1000) : null;

  return (
    <div className="media-lost" role="alertdialog" aria-live="assertive" aria-label={t.lostTitle}>
      <div className="media-lost-card">
        <div className="slot" aria-hidden="true">
          <span className="slot-card">
            <Icon name="card" />
          </span>
          <span className="slot-mouth" />
        </div>
        <h2>{t.lostTitle}</h2>
        <p>{t.lostBody}</p>
        <code dir="ltr">{offline.root}</code>
        {at && (
          <p className="resume-at">
            ▶ <span dir="ltr">{fmtClock(at)}</span>
          </p>
        )}
        <div className="waiting">
          <i />
          {t.waiting}
        </div>
        <div className="row">
          <button
            className="btn ghost"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              await disconnectMedia(false).catch(() => setBusy(false));
              useApp.getState().setNotice(null);
            }}
          >
            {t.openOtherFolder}
          </button>
        </div>
      </div>
    </div>
  );
}
