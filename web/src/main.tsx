import '@fontsource/karantina/hebrew-400.css';
import '@fontsource/karantina/hebrew-700.css';
import '@fontsource/karantina/latin-400.css';
import '@fontsource/karantina/latin-700.css';
import '@fontsource/ibm-plex-sans-hebrew/hebrew-400.css';
import '@fontsource/ibm-plex-sans-hebrew/hebrew-500.css';
import '@fontsource/ibm-plex-sans-hebrew/hebrew-600.css';
import '@fontsource/ibm-plex-sans-hebrew/latin-400.css';
import '@fontsource/ibm-plex-sans-hebrew/latin-500.css';
import '@fontsource/ibm-plex-sans-hebrew/latin-600.css';
import '@fontsource/martian-mono/latin-400.css';
import '@fontsource/martian-mono/latin-500.css';
import './styles/tokens.css';
import './styles/base.css';
import './styles/welcome.css';
import './styles/shell.css';
import './styles/library.css';
import './styles/stage.css';
import './styles/transport.css';
import './styles/inspector.css';
import './styles/overlays.css';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import { engine } from './player/engine';
import { useApp } from './store/app';

// handle for debugging and the browser smoke test (tests/e2e/smoke_chrome.py)
Object.assign(window, { dashcamStudio: { engine, store: useApp } });

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
