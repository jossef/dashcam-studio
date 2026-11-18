import type { Category, Clip, ClipDetail, Status, TrackPoint } from '../types';

const enc = encodeURIComponent;

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const r = await fetch(path, init);
  let body: unknown = {};
  try {
    body = await r.json();
  } catch {
    /* empty body */
  }
  if (!r.ok) throw new Error((body as { error?: string }).error || `${r.status} ${r.statusText}`);
  return body as T;
}

const post = <T,>(path: string, data: unknown = {}) =>
  request<T>(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });

export interface LibraryResponse {
  root: string | null;
  categories: Category[];
  clips: Omit<Clip, 'date'>[];
  hasGps: boolean;
}

export interface ExportJob {
  url: string;
  name: string;
  size: number;
  count: number;
}

export const api = {
  status: () => request<Status>('/api/status'),
  library: () => request<LibraryResponse>('/api/library'),
  clip: (id: string) => request<ClipDetail>(`/api/clip?id=${enc(id)}`),
  tracks: (ids: string[]) =>
    request<{ tracks: Record<string, TrackPoint[]> }>(`/api/tracks?${ids.map((i) => `id=${enc(i)}`).join('&')}`),
  open: (path: string) => post<Status>('/api/open', { path }),
  rescan: () => post<Status>('/api/rescan'),
  disconnect: (eject: boolean) =>
    post<{ root: string | null; released: boolean; ejected: boolean; message: string; status: Status }>('/api/disconnect', { eject }),
  drives: () => request<{ drives: { path: string; name: string }[] }>('/api/drives'),
  browse: (path: string) =>
    request<{ path: string; parent: string | null; dirs: { name: string; path: string }[]; isDashcam?: boolean; error?: string }>(
      `/api/browse?path=${enc(path)}`,
    ),
  favorites: () => request<{ ids: string[] }>('/api/favorites'),
  setFavorites: (ids: string[], on: boolean) => post<{ ids: string[] }>('/api/favorites', { ids, on }),
  createExport: (ids: string[], cams: string, gpx: boolean) => post<ExportJob>('/api/export', { ids, cams, gpx }),
};

export const urls = {
  media: (id: string, cam: string, t = 0) => `/media?id=${enc(id)}&cam=${cam}#t=${t.toFixed(3)}`,
  transcode: (id: string, cam: string, t = 0) => `/api/transcode?id=${enc(id)}&cam=${cam}&t=${t.toFixed(3)}`,
  download: (id: string, cam: string) => `/media?id=${enc(id)}&cam=${cam}&download=1`,
  thumb: (id: string, cam: string) => `/api/thumb?id=${enc(id)}&cam=${cam}`,
  zip: (ids: string[], gpx: boolean) => `/api/zip?${ids.map((i) => `id=${enc(i)}`).join('&')}${gpx ? '' : '&gpx=0'}`,
};

/** Hand a URL to the browser's download manager (keeps native progress + resume). */
export function startDownload(href: string) {
  const a = document.createElement('a');
  a.href = href;
  a.download = '';
  document.body.append(a);
  a.click();
  a.remove();
}
