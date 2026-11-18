import { useMemo, useState } from 'react';
import { camName, clipsN, t } from '../../i18n/he';
import { fmtBytes, fmtDateRange, sum } from '../../lib/format';
import { runExport } from '../../store/actions';
import { useApp } from '../../store/app';
import { CAMS, type Cam } from '../../types';
import { Modal } from '../common/Modal';
import { Icon } from '../Icon';

const GiB4 = 4 * 1024 ** 3;

/** Pick cameras + GPX for a multi-clip ZIP, see the size, then hand it to the browser. */
export function ExportModal({ ids }: { ids: string[] }) {
  const byId = useApp((s) => s.byId);
  const gpxPref = useApp((s) => s.gpx);
  const clips = useMemo(() => ids.map((id) => byId.get(id)).filter((c) => !!c), [ids, byId]);
  const [cams, setCams] = useState<Cam[]>(CAMS.filter((c) => clips.some((x) => x.cams[c])));
  const [gpx, setGpx] = useState(gpxPref);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const perCam = Object.fromEntries(CAMS.map((c) => [c, sum(clips, (x) => x.cams[c])])) as Record<Cam, number>;
  const counts = Object.fromEntries(CAMS.map((c) => [c, clips.filter((x) => x.cams[c]).length])) as Record<Cam, number>;
  const total = sum(cams, (c) => perCam[c]);
  const anyGps = clips.some((c) => c.gps);
  const dates = clips.map((c) => c.date).filter((d): d is Date => !!d);

  const go = async () => {
    setBusy(true);
    setError('');
    try {
      await runExport(ids, cams, gpx && anyGps);
      useApp.getState().setGpx(gpx);
      useApp.getState().openModal(null);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title={t.exportTitle(clips.length)}
      className="export-card"
      footer={
        <>
          <div className="export-total">
            <span className="eyebrow">{t.exportTotal}</span>
            <b dir="ltr">{fmtBytes(total)}</b>
          </div>
          <div className="row">
            <button className="btn ghost" onClick={() => useApp.getState().openModal(null)}>
              {t.cancel}
            </button>
            <button className="btn primary" disabled={!cams.length || busy} onClick={go}>
              <Icon name="zip" />
              {busy ? t.exportPreparing : t.exportGo}
            </button>
          </div>
        </>
      }
    >
      <div className="export-body">
        {dates.length > 0 && (
          <p className="export-span">
            <span className="eyebrow">{t.exportSpan}</span> {fmtDateRange(dates)} · {clipsN(clips.length)}
          </p>
        )}
        <span className="eyebrow">{t.exportCams}</span>
        <div className="cam-picks">
          {CAMS.map((c) => {
            const on = cams.includes(c);
            return (
              <button
                key={c}
                className={`cam-pick cam-${c} ${on ? 'on' : ''}`}
                disabled={!counts[c]}
                aria-pressed={on}
                onClick={() => setCams(on ? cams.filter((x) => x !== c) : CAMS.filter((x) => x === c || cams.includes(x)))}
              >
                <span className="dot" />
                <b>{camName[c]}</b>
                <small>
                  {counts[c] < clips.length ? `${counts[c]}/${clips.length} · ` : ''}
                  <span dir="ltr">{fmtBytes(perCam[c])}</span>
                </small>
                <span className="tick">{on && <Icon name="check" />}</span>
              </button>
            );
          })}
        </div>
        {!cams.length && <p className="error">{t.exportNoCams}</p>}
        <label className={`opt ${anyGps ? '' : 'disabled'}`}>
          <input type="checkbox" checked={gpx && anyGps} disabled={!anyGps} onChange={(e) => setGpx(e.target.checked)} />
          {t.exportGpx}
        </label>
        <p className="muted small">{clips.length > 1 ? t.exportLayout : ''}</p>
        {total > GiB4 && (
          <p className="note">
            <Icon name="warn" /> {t.exportBig}
          </p>
        )}
        {error && <p className="error">{error}</p>}
      </div>
    </Modal>
  );
}
