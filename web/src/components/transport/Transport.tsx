import { t } from '../../i18n/he';
import { useEngine } from '../../hooks/useEngine';
import { engine, RATES } from '../../player/engine';
import { go, neighbour, snapshot, toggleFavorite, toggleFullscreen } from '../../store/actions';
import { useApp } from '../../store/app';
import type { Layout } from '../../types';
import { Icon } from '../Icon';
import { Clock } from './Clock';
import { DownloadMenu } from './DownloadMenu';
import { Speedo } from './Speedo';
import { Timeline } from './Timeline';
import { TripRibbon } from './TripRibbon';
import { VolumeControl } from './VolumeControl';

const LAYOUTS: { id: Layout; icon: 'focus' | 'grid' | 'solo'; title: string }[] = [
  { id: 'focus', icon: 'focus', title: t.layoutFocus },
  { id: 'grid', icon: 'grid', title: t.layoutGrid },
  { id: 'solo', icon: 'solo', title: t.layoutSolo },
];

export function Transport() {
  const playing = useEngine((s) => s.playing);
  const rate = useEngine((s) => s.rate);
  const layout = useApp((s) => s.layout);
  const currentId = useApp((s) => s.current?.id);
  const starred = useApp((s) => !!s.current && s.favorites.has(s.current.id));
  // re-evaluated whenever the current clip changes
  const hasPrev = !!currentId && !!neighbour(-1);
  const hasNext = !!currentId && !!neighbour(1);

  return (
    <div className="transport">
      <TripRibbon />
      <Timeline />
      <div className="controls">
        {/* media controls keep their universal left-to-right order, also in RTL */}
        <div className="ctl-group playback" dir="ltr">
          <button className="icon-btn" title={t.prevClip} disabled={!hasPrev} onClick={() => go(-1)}>
            <Icon name="prev" />
          </button>
          <button className="icon-btn hide-sm" title={t.back10} onClick={() => engine.seek(engine.now() - 10)}>
            <Icon name="back" />
          </button>
          <button className="play-btn" title={t.playPause} onClick={() => engine.toggle()}>
            <Icon name={playing ? 'pause' : 'play'} />
          </button>
          <button className="icon-btn hide-sm" title={t.fwd10} onClick={() => engine.seek(engine.now() + 10)}>
            <Icon name="fwd" />
          </button>
          <button className="icon-btn" title={t.nextClip} disabled={!hasNext} onClick={() => go(1)}>
            <Icon name="next" />
          </button>
        </div>
        <Clock />
        <Speedo />
        <div className="spacer" />
        <div className="ctl-group">
          <button className={`icon-btn star-btn ${starred ? 'on' : ''}`} disabled={!currentId} title={starred ? t.unstar : t.star} onClick={() => toggleFavorite()}>
            <Icon name={starred ? 'starFill' : 'star'} />
          </button>
          <button className={`pill ${rate !== 1 ? 'hot' : ''}`} title={t.speed} dir="ltr" onClick={() => engine.setRate(RATES[(RATES.indexOf(rate) + 1) % RATES.length])}>
            {rate}×
          </button>
          <VolumeControl />
          <div className="seg" role="group">
            {LAYOUTS.map((l) => (
              <button key={l.id} className={layout === l.id ? 'active' : ''} title={l.title} onClick={() => useApp.getState().setLayout(l.id)}>
                <Icon name={l.icon} />
              </button>
            ))}
          </div>
          <button className="icon-btn snap-btn" title={t.snapshot} onClick={snapshot}>
            <Icon name="camera" />
          </button>
          <DownloadMenu />
          <button className="icon-btn" title={t.fullscreen} onClick={toggleFullscreen}>
            <Icon name="full" />
          </button>
        </div>
      </div>
    </div>
  );
}
