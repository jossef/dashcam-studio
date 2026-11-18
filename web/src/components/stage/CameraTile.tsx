import { useCallback, useRef } from 'react';
import { camName, t } from '../../i18n/he';
import { useEngine } from '../../hooks/useEngine';
import { engine } from '../../player/engine';
import { setMain, toggleFullscreen } from '../../store/actions';
import { useApp } from '../../store/app';
import type { Cam } from '../../types';
import { flash } from './BigFlash';

/**
 * One camera. The <video> element never leaves the DOM — layouts only move the
 * tile between grid slots, so swapping views is instant and keeps sync.
 */
export function CameraTile({ cam, slot }: { cam: Cam; slot: string }) {
  const snap = useEngine((s) => s.cams[cam]);
  const ffmpeg = useApp((s) => !!s.status?.ffmpeg);
  const hasClip = useApp((s) => !!s.current?.detail);
  const clickTimer = useRef<number>(undefined);
  const attach = useCallback((el: HTMLVideoElement | null) => engine.attach(cam, el), [cam]);
  const isMain = slot === 'main';

  let message = '';
  if (hasClip && snap.state === 'missing') message = t.notRecorded(cam);
  else if (snap.state === 'failed') message = ffmpeg ? t.playFailed(cam, snap.error || '') : t.noHevc;

  return (
    <figure
      className={`screen cam-${cam} ${isMain ? '' : 'side'} ${snap.state}`}
      data-slot={slot}
      onClick={() => {
        if (!isMain) return setMain(cam);
        clearTimeout(clickTimer.current);
        clickTimer.current = window.setTimeout(() => {
          engine.toggle();
          flash(engine.playing ? 'play' : 'pause');
        }, 220);
      }}
      onDoubleClick={(e) => {
        if (!isMain) return;
        e.preventDefault();
        clearTimeout(clickTimer.current);
        toggleFullscreen();
      }}
    >
      <video ref={attach} playsInline preload="auto" muted={cam !== 'F'} />
      <figcaption>
        <b>{camName[cam]}</b>
        <span className="res" dir="ltr">
          {snap.available ? snap.res : ''}
        </span>
      </figcaption>
      {snap.available && snap.mode === 'compat' && (
        <span className="compat" title={t.compatTitle}>
          {t.compatBadge}
        </span>
      )}
      {!isMain && <span className="side-hint">{t.makeMain}</span>}
      <div className="screen-state">{message}</div>
    </figure>
  );
}
