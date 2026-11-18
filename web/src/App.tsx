import { useEffect } from 'react';
import { Toasts } from './components/common/Toasts';
import { Inspector } from './components/inspector/Inspector';
import { LibraryPanel } from './components/library/LibraryPanel';
import { ExportModal } from './components/modals/ExportModal';
import { HelpModal } from './components/modals/HelpModal';
import { DisconnectModal } from './components/media/DisconnectModal';
import { MediaLostOverlay } from './components/media/MediaLostOverlay';
import { TopBar } from './components/shell/TopBar';
import { Stage } from './components/stage/Stage';
import { BrowseModal } from './components/welcome/BrowseModal';
import { WelcomeScreen } from './components/welcome/WelcomeScreen';
import { useKeyboard } from './hooks/useKeyboard';
import { engine } from './player/engine';
import { boot, checkConnection, go } from './store/actions';
import { useApp } from './store/app';

export function App() {
  const welcome = useApp((s) => s.welcome);
  const hasLibrary = useApp((s) => !!s.root);
  const modal = useApp((s) => s.modal);
  const showLibrary = useApp((s) => s.showLibrary);
  const showInspector = useApp((s) => s.showInspector);
  const grid = useApp((s) => s.layout === 'grid');
  const watching = useApp((s) => !!s.root || !!s.offline);
  useKeyboard();

  // notice a pulled / re-inserted card within ~2 s
  useEffect(() => {
    if (!watching) return;
    const id = window.setInterval(checkConnection, 2000);
    const onFocus = () => checkConnection();
    window.addEventListener('focus', onFocus);
    return () => {
      clearInterval(id);
      window.removeEventListener('focus', onFocus);
    };
  }, [watching]);

  useEffect(() => {
    boot();
    if ('mediaSession' in navigator) {
      navigator.mediaSession.setActionHandler('play', () => engine.play());
      navigator.mediaSession.setActionHandler('pause', () => engine.pause());
      navigator.mediaSession.setActionHandler('nexttrack', () => go(1));
      navigator.mediaSession.setActionHandler('previoustrack', () => go(-1));
    }
  }, []);

  return (
    <>
      {hasLibrary && (
        <div className={`app ${showLibrary ? '' : 'no-lib'} ${showInspector ? '' : 'no-insp'} ${grid ? 'grid-mode' : ''}`}>
          <TopBar />
          <LibraryPanel />
          <Stage />
          <Inspector />
          <MediaLostOverlay />
        </div>
      )}
      {welcome && <WelcomeScreen />}
      {modal?.kind === 'help' && <HelpModal />}
      {modal?.kind === 'browse' && <BrowseModal start={modal.start} />}
      {modal?.kind === 'export' && <ExportModal ids={modal.ids} />}
      {modal?.kind === 'disconnect' && <DisconnectModal />}
      <Toasts />
    </>
  );
}
