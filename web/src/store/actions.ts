/**
 * Cross-cutting actions that coordinate the store, the API and the playback engine.
 * Components call these; they never talk to the engine and API at the same time themselves.
 */
import { t } from '../i18n/he';
import { api, startDownload, urls } from '../lib/api';
import { clipBytes, dayKey, fmtBytes, fmtClock, sum } from '../lib/format';
import { gpsAt } from '../lib/gps';
import { storage } from '../lib/storage';
import { engine } from '../player/engine';
import { CAMS, type Cam, type Clip } from '../types';
import { isFavoritesView, useApp, type Resume } from './app';

const S = () => useApp.getState();
let loadToken = 0;

/** Chronological neighbour of the current clip within its category (playback order). */
export function neighbour(dir: 1 | -1): Clip | null {
  const { current, byId, clips } = S();
  const cur = current && byId.get(current.id);
  if (!cur) return null;
  const list = clips.filter((c) => c.category === cur.category);
  return list[list.findIndex((c) => c.id === cur.id) + dir] ?? null;
}

export async function openClip(id: string, { t: at = 0, play = false } = {}) {
  const st = S();
  const clip = st.byId.get(id);
  if (!clip) return;
  const token = ++loadToken;
  engine.pause();
  st.setCurrent({ id, detail: null });
  if (!isFavoritesView(st.view) && st.view !== clip.category) st.setView(clip.category);
  let detail;
  try {
    detail = await api.clip(id);
  } catch (e) {
    st.toast(t.openFailed((e as Error).message), { kind: 'warn' });
    return;
  }
  if (token !== loadToken) return;
  S().setCurrent({ id, detail });
  engine.load(id, detail, at, play);
  const main = S().main;
  if (!detail.cams[main]) S().setMain(CAMS.find((c) => detail.cams[c]) ?? 'F');
  history.replaceState(null, '', `#clip=${encodeURIComponent(id)}`);
  storage.set('last', { root: S().root, id });
  document.title = `${clip.date ? fmtClock(clip.date) : clip.stem} · ${t.appName}`;
  // warm the next clip so auto-advance is instant
  const nx = neighbour(1);
  if (nx) api.clip(nx.id).catch(() => {});
}

export function go(dir: 1 | -1) {
  const n = neighbour(dir);
  if (n) openClip(n.id, { play: engine.playing });
}

engine.onClipEnded = () => {
  const next = neighbour(1);
  if (next) openClip(next.id, { t: 0, play: true });
  else engine.pause();
};

let compatToastShown = false;
engine.onCompatFallback = () => {
  if (compatToastShown) return;
  compatToastShown = true;
  S().toast(t.compatSwitched, { sub: t.compatSwitchedSub, kind: 'warn', ms: 5200 });
};

export function setMain(cam: Cam) {
  const st = S();
  const d = st.current?.detail;
  if (d && !d.cams[cam]) {
    st.toast(t.notRecorded(cam));
    return;
  }
  st.setMain(cam);
}

export async function loadLibrary(resume?: Resume | null) {
  const lib = await api.library();
  const st = S();
  st.setLibrary(lib.root ?? '', lib.clips, lib.categories);
  await st.loadFavorites();
  const { byId, clips } = S();
  if (resume && byId.has(resume.id)) return openClip(resume.id, { t: resume.t, play: resume.play });
  // resume: #clip=… in the URL, else the last clip for this folder, else the newest clip
  const hashId = new URLSearchParams(location.hash.slice(1)).get('clip');
  const last = storage.get<{ root: string; id: string } | null>('last', null);
  const pick = (hashId && byId.get(hashId)) || (last?.root === lib.root && byId.get(last.id)) || clips[clips.length - 1];
  if (pick) await openClip(pick.id);
}

// ------------------------------------------------------------------ media connection
let checking = false;

/**
 * Compare the server's view of the card with ours. Called every couple of seconds
 * and immediately when a video fails to load.
 *  - card vanished → pause, remember the position, show the "disconnected" overlay
 *  - card is back  → reload the library and continue exactly where we were
 */
export async function checkConnection() {
  if (checking) return;
  checking = true;
  try {
    const status = await api.status();
    const st = S();
    st.setStatus(status);
    if (status.connection === 'lost' && st.root && !st.offline) {
      const resume = st.current ? { id: st.current.id, t: engine.now(), play: engine.playing } : null;
      engine.unload();
      st.setOffline({ root: status.lastRoot ?? st.root, resume });
      st.openModal(null);
    } else if (status.connection === 'connected' && st.offline && status.root) {
      const { resume } = st.offline;
      st.setOffline(null);
      await loadLibrary(resume);
      st.toast(t.reconnected, { sub: status.label || status.root });
    }
  } catch {
    /* server unreachable - try again on the next tick */
  } finally {
    checking = false;
  }
}

engine.onMediaError = () => void checkConnection();

/** User-initiated: release every file on the card, optionally eject the drive. */
export async function disconnectMedia(eject: boolean) {
  const st = S();
  engine.unload(); // browsers drop their open video connections
  const label = st.status?.label || st.root || '';
  const res = await api.disconnect(eject);
  st.setStatus(res.status);
  st.setOffline(null);
  st.resetLibrary();
  st.openModal(null);
  history.replaceState(null, '', location.pathname);
  document.title = t.appName;
  if (!eject) st.setNotice({ kind: 'ok', title: t.disconnectedTitle, body: label });
  else if (res.ejected) st.setNotice({ kind: 'ok', title: t.safeToRemove, body: label });
  else st.setNotice({ kind: 'warn', title: t.ejectFailed, body: res.message });
  st.setWelcome(true);
}

export async function openRoot(path: string) {
  const status = await api.open(path); // throws → caller shows the error
  const st = S();
  st.setStatus(status);
  st.setOffline(null);
  st.setNotice(null);
  const rec = storage.get<string[]>('recents', []);
  storage.set('recents', [status.root!, ...rec.filter((x) => x !== status.root)].slice(0, 6));
  engine.unload();
  st.setCurrent(null);
  history.replaceState(null, '', location.pathname);
  await loadLibrary();
  st.setWelcome(false);
}

export async function rescan() {
  const st = S();
  try {
    await api.rescan();
    await loadLibrary();
    st.toast(t.rescanned, { sub: `${S().clips.length}` });
  } catch (e) {
    st.toast((e as Error).message, { kind: 'warn' });
  }
}

let booted = false;
export async function boot() {
  if (booted) return; // StrictMode runs effects twice in development
  booted = true;
  const st = S();
  let status;
  try {
    status = await api.status();
  } catch {
    status = { root: null, ffmpeg: false } as never;
  }
  st.setStatus(status);
  engine.ffmpeg = !!status.ffmpeg;
  engine.decoder = st.decoder;
  engine.volume = storage.get('volume', 0.8);
  engine.muted = storage.get('muted', false);
  if (status.root) {
    await loadLibrary();
    if (status.connection === 'lost') await checkConnection(); // reloaded while the card was out
  } else st.setWelcome(true);
}

export function persistVolume() {
  storage.set('volume', engine.volume);
  storage.set('muted', engine.muted);
}

// ------------------------------------------------------------------ favorites
export function toggleFavorite(id?: string) {
  const st = S();
  const cid = id ?? st.current?.id;
  if (!cid) return;
  const on = !st.favorites.has(cid);
  st.setFavorite([cid], on);
  if (!id) st.toast(on ? t.starred : t.unstarred);
}

// ------------------------------------------------------------------ downloads
export function downloadCurrentZip() {
  const st = S();
  const c = st.current && st.byId.get(st.current.id);
  if (!c) return;
  startDownload(urls.zip([c.id], st.gpx));
  st.toast(t.downloadingCams(Object.keys(c.cams).length), { sub: fmtBytes(clipBytes(c)) });
}

export function openExport(ids: string[]) {
  if (ids.length) S().openModal({ kind: 'export', ids });
}

export async function runExport(ids: string[], cams: Cam[], gpx: boolean) {
  const st = S();
  const job = await api.createExport(ids, cams.join(''), gpx);
  startDownload(job.url);
  st.toast(t.exportStarted(job.count), { sub: `${job.name} · ${fmtBytes(job.size)}`, ms: 5000 });
}

export function selectionBytes(ids: Iterable<string>, cams: Cam[] = CAMS) {
  const { byId } = S();
  return sum([...ids], (id) => sum(cams, (c) => byId.get(id)?.cams[c]));
}

// ------------------------------------------------------------------ snapshot
export function snapshot() {
  const st = S();
  const clip = st.current && st.byId.get(st.current.id);
  if (!clip) return;
  const vids = engine.videoElements();
  if (!vids.length) return;
  const main = vids.find((v) => v.cam === st.main) ?? vids[0];
  const others = vids.filter((v) => v !== main);
  const W = 1920;
  const h = (el: HTMLVideoElement, w: number) => Math.round((w * el.videoHeight) / el.videoWidth);
  const mh = h(main.el, W);
  const ow = others.length ? W / others.length : 0;
  const oh = others.length ? Math.max(...others.map((v) => h(v.el, ow))) : 0;
  const bar = 56;
  const cv = document.createElement('canvas');
  cv.width = W;
  cv.height = mh + oh + bar;
  const g = cv.getContext('2d')!;
  g.fillStyle = '#0a0c0f';
  g.fillRect(0, 0, cv.width, cv.height);
  try {
    g.drawImage(main.el, 0, 0, W, mh);
    others.forEach((v, i) => g.drawImage(v.el, i * ow, mh, ow, h(v.el, ow)));
  } catch (e) {
    st.toast(t.snapFailed, { sub: (e as Error).message, kind: 'warn' });
    return;
  }
  const now = engine.now();
  const when = clip.date ? new Date(clip.date.getTime() + now * 1000) : null;
  const gp = gpsAt(st.current?.detail?.track, now);
  g.fillStyle = '#ffb238';
  g.fillRect(W - 8, mh + oh, 8, bar);
  g.fillStyle = '#e9e5dc';
  g.font = '500 24px "Martian Mono", monospace';
  g.textBaseline = 'middle';
  const text = [when ? `${dayKey(when)} ${fmtClock(when)}` : clip.stem, gp ? `${Math.round(gp.kmh)} km/h` : '', gp ? `${gp.lat.toFixed(5)}, ${gp.lon.toFixed(5)}` : '']
    .filter(Boolean)
    .join('   ·   ');
  g.fillText(text, 28, mh + oh + bar / 2);
  cv.toBlob((blob) => {
    if (!blob) return;
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `snapshot_${when ? `${dayKey(when)}_${fmtClock(when).replace(/:/g, '-')}` : clip.stem}.png`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 4000);
    st.toast(t.snapSaved, { sub: t.snapCams(vids.length) });
  }, 'image/png');
}

export function toggleFullscreen() {
  if (document.fullscreenElement) document.exitFullscreen();
  else document.getElementById('stage')?.requestFullscreen?.().catch(() => {});
}
