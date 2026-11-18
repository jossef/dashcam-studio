import { useEffect, useRef, useState } from 'react';
import { t } from '../../i18n/he';
import { useApp } from '../../store/app';
import { CAMS } from '../../types';
import { MapView } from '../inspector/MapView';
import { Transport } from '../transport/Transport';
import { BigFlash } from './BigFlash';
import { CameraTile } from './CameraTile';

export function Stage() {
  const layout = useApp((s) => s.layout);
  const main = useApp((s) => s.main);
  const hasClip = useApp((s) => !!s.current);
  const order = [main, ...CAMS.filter((c) => c !== main)];
  const [idle, setIdle] = useState(false);
  const idleTimer = useRef<number>(undefined);

  // fullscreen: hide the controls after a moment without mouse movement
  const onMove = () => {
    setIdle(false);
    clearTimeout(idleTimer.current);
    idleTimer.current = window.setTimeout(() => document.fullscreenElement && setIdle(true), 2200);
  };
  useEffect(() => () => clearTimeout(idleTimer.current), []);

  return (
    <main className={`stage ${idle ? 'idle' : ''}`} id="stage" onPointerMove={onMove}>
      <div className={`screens layout-${layout}`}>
        {CAMS.map((c) => (
          <CameraTile key={c} cam={c} slot={order.indexOf(c) === 0 ? 'main' : `s${order.indexOf(c)}`} />
        ))}
        {layout === 'grid' && (
          <div className="screen map-slot">
            <MapView />
          </div>
        )}
        <BigFlash />
        {!hasClip && <div className="stage-empty">{t.pickClip}</div>}
      </div>
      <Transport />
    </main>
  );
}
