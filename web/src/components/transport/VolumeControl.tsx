import { t } from '../../i18n/he';
import { useEngine } from '../../hooks/useEngine';
import { engine } from '../../player/engine';
import { persistVolume } from '../../store/actions';
import { Icon } from '../Icon';

export function VolumeControl() {
  const volume = useEngine((s) => s.volume);
  const muted = useEngine((s) => s.muted);
  return (
    <div className="vol">
      <button
        className="icon-btn"
        title={t.mute}
        onClick={() => {
          engine.toggleMute();
          persistVolume();
        }}
      >
        <Icon name={muted || volume === 0 ? 'mute' : 'vol'} />
      </button>
      <input
        type="range"
        min={0}
        max={1}
        step={0.05}
        dir="ltr"
        aria-label={t.volume}
        value={muted ? 0 : volume}
        onChange={(e) => {
          engine.setVolume(+e.target.value);
          persistVolume();
        }}
      />
    </div>
  );
}
