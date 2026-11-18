import { useEffect, useRef, useState } from 'react';
import { clipsN, t } from '../../i18n/he';
import { urls } from '../../lib/api';
import { clipBytes, fmtBytes } from '../../lib/format';
import { downloadCurrentZip, openExport } from '../../store/actions';
import { useApp } from '../../store/app';
import { CAMS } from '../../types';
import { Icon } from '../Icon';

export function DownloadMenu() {
  const [open, setOpen] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);
  const clip = useApp((s) => (s.current ? s.byId.get(s.current.id) : undefined));
  const trip = useApp((s) => (s.current ? s.tripOf.get(s.current.id) : undefined));
  const gpx = useApp((s) => s.gpx);

  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent | KeyboardEvent) => {
      if (e instanceof KeyboardEvent ? e.key === 'Escape' : !wrap.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', close);
    document.addEventListener('keydown', close);
    return () => {
      document.removeEventListener('mousedown', close);
      document.removeEventListener('keydown', close);
    };
  }, [open]);

  const n = clip ? Object.keys(clip.cams).length : 0;
  const { toast, setGpx } = useApp.getState();

  return (
    <div className="dl-wrap" ref={wrap}>
      <button className="btn primary dl-btn" title={`${t.download} (D)`} disabled={!clip} onClick={() => setOpen(!open)}>
        <Icon name="zip" />
        <span>{t.download}</span>
      </button>
      {open && clip && (
        <div className="menu" role="menu">
          <button
            className="hero"
            onClick={() => {
              downloadCurrentZip();
              setOpen(false);
            }}
          >
            <Icon name="zip" />
            <span>
              {t.allCams(n)} <small>· ZIP{clip.gps && gpx ? ' + GPX' : ''}</small>
            </span>
            <small>{fmtBytes(clipBytes(clip))}</small>
          </button>
          <hr />
          {CAMS.filter((k) => clip.cams[k]).map((k) => (
            <a
              key={k}
              href={urls.download(clip.id, k)}
              download
              onClick={() => {
                toast(t.downloadingCam(k));
                setOpen(false);
              }}
            >
              <span className={`dot cam-${k}`} />
              <span>
                {t.camOnly(k)} <small>· MP4</small>
              </span>
              <small>{fmtBytes(clip.cams[k] || 0)}</small>
            </a>
          ))}
          {trip && trip.clips.length > 1 && (
            <>
              <hr />
              <button
                onClick={() => {
                  openExport(trip.clips.map((c) => c.id));
                  setOpen(false);
                }}
              >
                <Icon name="download" />
                <span>
                  {t.wholeTrip} <small>· {clipsN(trip.clips.length)}</small>
                </span>
                <small>{fmtBytes(trip.bytes)}</small>
              </button>
            </>
          )}
          <hr />
          <label>
            <input type="checkbox" checked={gpx} onChange={(e) => setGpx(e.target.checked)} /> {t.includeGpx}
          </label>
        </div>
      )}
    </div>
  );
}

