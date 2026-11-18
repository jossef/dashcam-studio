import { create } from 'zustand';
import { t } from '../i18n/he';
import { api } from '../lib/api';
import { buildTrips, chronological } from '../lib/library';
import { storage } from '../lib/storage';
import { FAVORITES_VIEW, type Cam, type Category, type Clip, type ClipDetail, type Decoder, type Layout, type Status, type Trip } from '../types';

export interface Toast {
  id: number;
  msg: string;
  sub?: string;
  kind?: 'warn' | '';
}

export type Modal = null | { kind: 'help' } | { kind: 'disconnect' } | { kind: 'browse'; start: string } | { kind: 'export'; ids: string[] };

/** Where to pick up again once a pulled card is plugged back in. */
export interface Resume {
  id: string;
  t: number;
  play: boolean;
}

export interface Notice {
  kind: 'ok' | 'warn';
  title: string;
  body?: string;
}

interface Current {
  id: string;
  detail: ClipDetail | null;
}

export interface AppState {
  status: Status | null;
  root: string | null;
  clips: Clip[]; // chronological
  byId: Map<string, Clip>;
  categories: Category[];
  /** trips per category, newest first */
  trips: Map<string, Trip[]>;
  tripOf: Map<string, Trip>;
  /** category name, or FAVORITES_VIEW */
  view: string | null;
  current: Current | null;

  layout: Layout;
  main: Cam;
  showLibrary: boolean;
  showInspector: boolean;
  decoder: Decoder;
  gpx: boolean;
  welcome: boolean;
  modal: Modal;
  toasts: Toast[];

  /** set while the card is unplugged; the player waits for it to come back */
  offline: { root: string; resume: Resume | null } | null;
  /** message shown on the welcome screen (e.g. "safe to remove") */
  notice: Notice | null;

  favorites: Set<string>;
  selection: Set<string>;
  /** anchor for shift-click range selection */
  anchor: string | null;

  // actions
  setStatus(s: Status): void;
  setLibrary(root: string, clips: Omit<Clip, 'date'>[], categories: Category[]): void;
  resetLibrary(): void;
  setOffline(o: AppState['offline']): void;
  setNotice(n: Notice | null): void;
  setView(v: string): void;
  setCurrent(c: Current | null): void;
  setLayout(l: Layout): void;
  cycleLayout(): void;
  setMain(c: Cam): void;
  togglePanel(p: 'showLibrary' | 'showInspector'): void;
  setDecoder(d: Decoder): void;
  setGpx(v: boolean): void;
  setWelcome(v: boolean): void;
  openModal(m: Modal): void;
  toast(msg: string, opts?: { sub?: string; kind?: 'warn' | ''; ms?: number }): void;

  loadFavorites(): Promise<void>;
  setFavorite(ids: string[], on: boolean): Promise<void>;
  toggleSelect(id: string, opts?: { range?: string[] }): void;
  setSelected(ids: string[], on: boolean): void;
  clearSelection(): void;
}

let toastSeq = 0;

export const useApp = create<AppState>((set, get) => ({
  status: null,
  root: null,
  clips: [],
  byId: new Map(),
  categories: [],
  trips: new Map(),
  tripOf: new Map(),
  view: null,
  current: null,

  layout: storage.get<Layout>('layout', 'focus'),
  main: 'F',
  showLibrary: storage.get('showLibrary', true),
  showInspector: storage.get('showInspector', true),
  decoder: storage.get<Decoder>('decoder', 'auto'),
  gpx: storage.get('gpx', true),
  welcome: false,
  modal: null,
  toasts: [],

  offline: null,
  notice: null,

  favorites: new Set(),
  selection: new Set(),
  anchor: null,

  setStatus: (status) => set({ status }),

  setLibrary(root, raw, categories) {
    const clips = chronological(raw.map((c) => ({ ...c, date: c.start ? new Date(c.start) : null })));
    const byId = new Map(clips.map((c) => [c.id, c]));
    const trips = new Map<string, Trip[]>();
    const tripOf = new Map<string, Trip>();
    for (const cat of categories) {
      const list = buildTrips(clips.filter((c) => c.category === cat.name));
      trips.set(cat.name, list);
      for (const tr of list) for (const c of tr.clips) tripOf.set(c.id, tr);
    }
    const saved = storage.get<string | null>('view', null);
    const view =
      saved === FAVORITES_VIEW
        ? saved
        : categories.find((c) => c.name === saved && c.count)?.name ?? categories.find((c) => c.count)?.name ?? null;
    set({ root, clips, byId, categories, trips, tripOf, view, selection: new Set(), anchor: null });
  },

  resetLibrary: () =>
    set({ root: null, clips: [], byId: new Map(), categories: [], trips: new Map(), tripOf: new Map(), current: null, selection: new Set(), anchor: null }),
  setOffline: (offline) => set({ offline }),
  setNotice: (notice) => set({ notice }),

  setView(view) {
    storage.set('view', view);
    set({ view, anchor: null });
  },
  setCurrent: (current) => set({ current }),
  setLayout(layout) {
    storage.set('layout', layout);
    set({ layout });
  },
  cycleLayout() {
    const L: Layout[] = ['focus', 'grid', 'solo'];
    get().setLayout(L[(L.indexOf(get().layout) + 1) % L.length]);
  },
  setMain: (main) => set({ main }),
  togglePanel(p) {
    const v = !get()[p];
    storage.set(p, v);
    set({ [p]: v } as Partial<AppState>);
  },
  setDecoder(decoder) {
    storage.set('decoder', decoder);
    set({ decoder });
  },
  setGpx(gpx) {
    storage.set('gpx', gpx);
    set({ gpx });
  },
  setWelcome: (welcome) => set({ welcome }),
  openModal: (modal) => set({ modal }),

  toast(msg, { sub, kind = '', ms = 3200 } = {}) {
    const id = ++toastSeq;
    set({ toasts: [...get().toasts, { id, msg, sub, kind }] });
    setTimeout(() => set({ toasts: get().toasts.filter((x) => x.id !== id) }), ms);
  },

  async loadFavorites() {
    try {
      const { ids } = await api.favorites();
      set({ favorites: new Set(ids) });
    } catch {
      /* favorites are optional */
    }
  },

  async setFavorite(ids, on) {
    const prev = get().favorites;
    const next = new Set(prev);
    for (const id of ids) (on ? next.add(id) : next.delete(id));
    set({ favorites: next }); // optimistic
    try {
      const res = await api.setFavorites(ids, on);
      set({ favorites: new Set(res.ids) });
    } catch (e) {
      set({ favorites: prev });
      get().toast(t.favSaveFailed, { sub: (e as Error).message, kind: 'warn' });
    }
  },

  toggleSelect(id, { range } = {}) {
    const { selection, anchor } = get();
    const next = new Set(selection);
    if (range && anchor && range.includes(anchor) && range.includes(id)) {
      const [a, b] = [range.indexOf(anchor), range.indexOf(id)].sort((x, y) => x - y);
      const on = !selection.has(id) || selection.has(anchor);
      for (const x of range.slice(a, b + 1)) (on ? next.add(x) : next.delete(x));
    } else if (next.has(id)) next.delete(id);
    else next.add(id);
    set({ selection: next, anchor: id });
  },

  setSelected(ids, on) {
    const next = new Set(get().selection);
    for (const id of ids) (on ? next.add(id) : next.delete(id));
    set({ selection: next });
  },

  clearSelection: () => set({ selection: new Set(), anchor: null }),
}));

export const isFavoritesView = (v: string | null) => v === FAVORITES_VIEW;
