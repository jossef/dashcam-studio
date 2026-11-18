/**
 * Multi-camera playback engine — framework-agnostic.
 *
 * Front is the master clock (or the first camera that is available). Slave
 * cameras follow it: small drift is absorbed by nudging playbackRate
 * (invisible), large drift triggers a seek, and if any camera buffers the
 * others wait for it. React components only attach <video> elements and read
 * a snapshot through `useEngine()`.
 */
import { urls } from '../lib/api';
import { clamp } from '../lib/format';
import { CAMS, type Cam, type CamInfo, type ClipDetail, type Decoder } from '../types';

export type CamState = 'idle' | 'ready' | 'buffering' | 'missing' | 'failed';
export type CamMode = 'native' | 'compat';

export interface CamSnapshot {
  available: boolean;
  mode: CamMode;
  state: CamState;
  error?: string;
  res: string;
}

export interface EngineSnapshot {
  clipId: string | null;
  playing: boolean;
  rate: number;
  volume: number;
  muted: boolean;
  master: Cam;
  cams: Record<Cam, CamSnapshot>;
}

export const RATES = [0.25, 0.5, 1, 1.5, 2, 4];

class CamPlayer {
  el: HTMLVideoElement | null = null;
  info: CamInfo | null = null;
  available = false;
  mode: CamMode = 'native';
  base = 0; // compat streams start at `base` seconds
  dur = 0;
  failed = false;
  gated = false;
  state: CamState = 'idle';
  error?: string;
  private pendingSrc: string | null = null;

  constructor(readonly cam: Cam, private readonly engine: PlaybackEngine) {}

  get time() {
    if (!this.el) return 0;
    return this.mode === 'native' ? this.el.currentTime : this.base + (this.el.currentTime || 0);
  }

  get ready() {
    return !this.available || this.failed || !this.el || this.el.ended || this.el.readyState >= 3;
  }

  get active() {
    return this.available && !this.failed && !!this.el;
  }

  attach(el: HTMLVideoElement | null) {
    if (this.el === el) return;
    this.el = el;
    if (!el) return;
    el.addEventListener('error', () => this.onError());
    el.addEventListener('waiting', () => this.setState('buffering'));
    for (const ev of ['playing', 'canplay', 'seeked', 'loadeddata']) el.addEventListener(ev, () => this.available && !this.failed && this.setState('ready'));
    el.addEventListener('ended', () => this.engine.onCamEnded(this));
    if (this.pendingSrc) {
      el.src = this.pendingSrc;
      this.pendingSrc = null;
    }
    this.engine.applyVolume();
  }

  private setState(s: CamState, error?: string) {
    if (this.state === s && this.error === error) return;
    this.state = s;
    this.error = error;
    this.engine.emit();
  }

  private setSrc(src: string | null) {
    if (!this.el) {
      this.pendingSrc = src;
      return;
    }
    if (src) this.el.src = src;
    else {
      this.el.removeAttribute('src');
      this.el.load();
    }
  }

  url(t: number) {
    const id = this.engine.clipId!;
    return this.mode === 'native' ? urls.media(id, this.cam, t) : urls.transcode(id, this.cam, t);
  }

  load(info: CamInfo | undefined, t: number) {
    this.failed = false;
    this.gated = false;
    this.info = info ?? null;
    this.available = !!info;
    if (!info) {
      this.setSrc(null);
      this.setState('missing');
      return;
    }
    this.dur = info.duration || 0;
    this.mode = this.engine.chooseMode(info);
    this.base = this.mode === 'compat' ? t : 0;
    this.setState('buffering');
    this.setSrc(this.url(t));
  }

  seek(t: number) {
    if (!this.available || this.failed || !this.el) return;
    t = clamp(t, 0, Math.max(0, (this.dur || 1e9) - 0.05));
    if (this.mode === 'native') {
      if (Math.abs(this.el.currentTime - t) > 0.001) this.el.currentTime = t;
    } else {
      this.base = t;
      this.el.src = this.url(t);
      if (this.engine.playing) this.el.play().catch(() => {});
    }
  }

  private onError() {
    if (!this.available || !this.el?.getAttribute('src')) return;
    const err = this.el.error;
    const e = this.engine;
    e.onMediaError?.();
    if (this.mode === 'native' && e.ffmpeg && e.decoder !== 'native') {
      const t = this.time || 0;
      this.mode = 'compat';
      this.base = t;
      this.el.src = this.url(t);
      if (e.playing) this.el.play().catch(() => {});
      e.onCompatFallback?.();
      e.emit();
      return;
    }
    this.failed = true;
    this.setState('failed', err?.message || 'decode error');
  }

  snapshot(): CamSnapshot {
    const i = this.info;
    const res = i?.height ? `${i.height >= 2160 ? '4K' : `${i.height}p`} · ${Math.round(i.fps || 0)}fps` : '';
    return { available: this.available, mode: this.mode, state: this.state, error: this.error, res };
  }
}

export class PlaybackEngine {
  readonly players: Record<Cam, CamPlayer>;
  clipId: string | null = null;
  playing = false;
  rate = 1;
  volume = 0.8;
  muted = false;
  ffmpeg = false;
  decoder: Decoder = 'auto';
  /** Called when the master camera reaches the end while playing. */
  onClipEnded?: () => void;
  onCompatFallback?: () => void;
  /** A camera failed to load — maybe the card was pulled. */
  onMediaError?: () => void;

  private listeners = new Set<() => void>();
  private snap: EngineSnapshot;
  private timer: number | undefined;

  constructor() {
    this.players = { F: new CamPlayer('F', this), R: new CamPlayer('R', this), C: new CamPlayer('C', this) };
    this.snap = this.buildSnapshot();
  }

  // ------------------------------------------------------------ subscription
  subscribe = (fn: () => void) => {
    this.listeners.add(fn);
    if (this.timer === undefined && typeof window !== 'undefined') this.timer = window.setInterval(() => this.tick(), 200);
    return () => this.listeners.delete(fn);
  };
  getSnapshot = () => this.snap;
  emit() {
    this.snap = this.buildSnapshot();
    this.listeners.forEach((fn) => fn());
  }
  private buildSnapshot(): EngineSnapshot {
    return {
      clipId: this.clipId,
      playing: this.playing,
      rate: this.rate,
      volume: this.volume,
      muted: this.muted,
      master: this.master().cam,
      cams: { F: this.players.F.snapshot(), R: this.players.R.snapshot(), C: this.players.C.snapshot() },
    };
  }

  // ----------------------------------------------------------------- basics
  attach(cam: Cam, el: HTMLVideoElement | null) {
    this.players[cam].attach(el);
  }
  master(): CamPlayer {
    return this.players[CAMS.find((c) => this.players[c].available && !this.players[c].failed) || 'F'];
  }
  activePlayers() {
    return CAMS.map((c) => this.players[c]).filter((p) => p.active);
  }
  now() {
    return this.master().time || 0;
  }
  duration() {
    const m = this.master();
    if (m.dur) return m.dur;
    return m.el && Number.isFinite(m.el.duration) ? m.el.duration : 60;
  }
  bufferedEnd(t: number) {
    const m = this.master();
    const b = m.el?.buffered;
    if (!b) return 0;
    if (m.mode !== 'native') return m.base + (b.length ? b.end(b.length - 1) : 0);
    let end = 0;
    for (let i = 0; i < b.length; i++) if (b.start(i) <= t + 0.5) end = Math.max(end, b.end(i));
    return end;
  }
  videoElements() {
    return this.activePlayers()
      .filter((p) => p.el!.videoWidth)
      .map((p) => ({ cam: p.cam, el: p.el! }));
  }

  chooseMode(info: CamInfo): CamMode {
    if (this.decoder === 'compat' && this.ffmpeg) return 'compat';
    if (this.decoder === 'native' || !info.mime || !this.ffmpeg) return 'native';
    const type = `video/mp4; codecs="${info.mime}"`;
    const probe = document.createElement('video').canPlayType(type);
    const ms = (window as { MediaSource?: { isTypeSupported?: (t: string) => boolean } }).MediaSource?.isTypeSupported?.(type);
    return probe || ms ? 'native' : 'compat';
  }

  // --------------------------------------------------------------- transport
  load(clipId: string, detail: ClipDetail, t: number, autoplay: boolean) {
    this.pauseElements();
    this.playing = false;
    this.clipId = clipId;
    for (const c of CAMS) this.players[c].load(detail.cams[c], t);
    this.applyVolume();
    this.setRate(this.rate);
    if (autoplay) this.play();
    else this.emit();
  }

  unload() {
    this.pauseElements();
    this.playing = false;
    this.clipId = null;
    for (const c of CAMS) this.players[c].load(undefined, 0);
    this.emit();
  }

  private pauseElements() {
    for (const c of CAMS) this.players[c].el?.pause();
  }

  play() {
    if (!this.clipId) return;
    this.playing = true;
    const m = this.master();
    if (m.el?.ended) this.seek(0);
    for (const p of this.activePlayers()) {
      if (p.el!.ended && this.now() < p.dur - 0.3) p.seek(this.now());
      if (!p.el!.ended || p === m) p.el!.play().catch(() => {});
    }
    this.emit();
  }

  pause() {
    this.playing = false;
    for (const p of this.activePlayers()) {
      p.el!.pause();
      p.gated = false;
    }
    this.emit();
  }

  toggle() {
    if (this.playing) this.pause();
    else this.play();
  }

  seek(t: number) {
    if (!this.clipId) return;
    t = clamp(t, 0, this.duration() - 0.05);
    for (const p of this.activePlayers()) p.seek(t);
  }

  step(frames: number) {
    const fps = this.master().info?.fps || 25;
    this.pause();
    this.seek(this.now() + frames / fps);
  }

  setRate(r: number) {
    this.rate = r;
    for (const p of this.activePlayers()) p.el!.playbackRate = r;
    this.emit();
  }

  setVolume(v: number, muted = v === 0) {
    this.volume = clamp(v, 0, 1);
    this.muted = muted;
    this.applyVolume();
    this.emit();
  }

  toggleMute() {
    this.muted = !this.muted;
    this.applyVolume();
    this.emit();
  }

  /** Only the master camera is audible. */
  applyVolume() {
    const m = this.master();
    for (const c of CAMS) {
      const el = this.players[c].el;
      if (!el) continue;
      el.muted = this.players[c] !== m || this.muted;
      el.volume = this.volume;
    }
  }

  onCamEnded(p: CamPlayer) {
    if (p === this.master() && this.playing) {
      if (this.onClipEnded) this.onClipEnded();
      else this.pause();
    }
  }

  // --------------------------------------------------------------- sync loop
  private tick() {
    if (!this.clipId) return;
    const m = this.master();
    const mt = m.time;
    const ps = this.activePlayers();
    if (this.playing) {
      const stalled = ps.filter((p) => !p.ready);
      if (stalled.length) {
        for (const p of ps)
          if (!stalled.includes(p) && !p.el!.paused) {
            p.el!.pause();
            p.gated = true;
          }
      } else {
        for (const p of ps)
          if (p.gated || (p.el!.paused && !p.el!.ended)) {
            p.gated = false;
            p.el!.play().catch(() => {});
          }
      }
    }
    for (const p of ps) {
      const el = p.el!;
      if (p === m || el.seeking) continue;
      if (el.ended) {
        if (mt < p.dur - 0.5) p.seek(mt);
        continue;
      }
      if (p.dur && mt > p.dur) continue;
      const diff = p.time - mt;
      if (p.mode === 'compat' || m.mode === 'compat') {
        if (Math.abs(diff) > 1.5) p.seek(mt);
        continue;
      }
      if (Math.abs(diff) > 0.35 || !this.playing) {
        if (Math.abs(diff) > (this.playing ? 0.35 : 0.02)) p.seek(mt + (this.playing ? 0.04 : 0));
        el.playbackRate = this.rate;
      } else if (Math.abs(diff) > 0.03) {
        el.playbackRate = this.rate * (1 - clamp(diff, -0.3, 0.3) * 0.5);
      } else {
        el.playbackRate = this.rate;
      }
    }
  }
}

export const engine = new PlaybackEngine();
