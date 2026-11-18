import { useEffect } from 'react';
import { clamp } from '../lib/format';
import { engine, RATES } from '../player/engine';
import { downloadCurrentZip, go, persistVolume, setMain, snapshot, toggleFavorite, toggleFullscreen } from '../store/actions';
import { useApp } from '../store/app';
import { flash } from '../components/stage/BigFlash';

/**
 * Global shortcuts. Letters are matched by physical key (`e.code`), so they
 * work the same with a Hebrew keyboard layout active.
 */
export function useKeyboard() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (target.closest('input, textarea, select, [contenteditable]') || e.ctrlKey || e.metaKey || e.altKey) return;
      const st = useApp.getState();
      if (e.key === 'Escape') {
        if (st.modal) st.openModal(null);
        else if (st.selection.size) st.clearSelection();
        else if (st.welcome && st.clips.length) st.setWelcome(false);
        return;
      }
      if (st.welcome || st.modal || st.offline) return;
      // keyboard-focused controls handle Space/Enter themselves; after a mouse click Space means play/pause
      if ((e.key === ' ' || e.key === 'Enter') && target.closest('button, [role="checkbox"], a') && target.matches(':focus-visible')) return;

      const now = engine.now();
      const toggle = () => {
        engine.toggle();
        flash(engine.playing ? 'play' : 'pause');
      };
      const byCode: Record<string, () => void> = {
        Space: toggle,
        KeyK: toggle,
        ArrowLeft: () => engine.seek(now - (e.shiftKey ? 1 : 5)),
        ArrowRight: () => engine.seek(now + (e.shiftKey ? 1 : 5)),
        KeyJ: () => engine.seek(now - 10),
        KeyL: () => engine.seek(now + 10),
        Comma: () => engine.step(-1),
        Period: () => engine.step(1),
        KeyP: () => go(-1),
        KeyN: () => go(1),
        PageUp: () => go(-1),
        PageDown: () => go(1),
        Digit1: () => setMain('F'),
        Digit2: () => setMain('R'),
        Digit3: () => setMain('C'),
        KeyG: () => st.cycleLayout(),
        KeyF: toggleFullscreen,
        KeyM: () => (engine.toggleMute(), persistVolume()),
        ArrowUp: () => (engine.setVolume(clamp(engine.volume + 0.1, 0, 1), false), persistVolume()),
        ArrowDown: () => (engine.setVolume(clamp(engine.volume - 0.1, 0, 1)), persistVolume()),
        BracketLeft: () => engine.setRate(RATES[Math.max(0, RATES.indexOf(engine.rate) - 1)]),
        BracketRight: () => engine.setRate(RATES[Math.min(RATES.length - 1, RATES.indexOf(engine.rate) + 1)]),
        KeyS: snapshot,
        KeyD: downloadCurrentZip,
        KeyB: () => st.togglePanel('showLibrary'),
        KeyI: () => st.togglePanel('showInspector'),
        KeyE: () => st.openModal({ kind: 'disconnect' }),
        Home: () => engine.seek(0),
        End: () => engine.seek(engine.duration() - 0.5),
        NumpadMultiply: () => toggleFavorite(),
      };
      let fn = e.key === '*' ? () => toggleFavorite() : e.key === '?' ? () => st.openModal({ kind: 'help' }) : byCode[e.code];
      if (e.code === 'Digit8' && e.shiftKey) fn = () => toggleFavorite();
      if (e.code === 'Slash' && e.shiftKey) fn = () => st.openModal({ kind: 'help' });
      if (fn) {
        e.preventDefault();
        fn();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);
}
