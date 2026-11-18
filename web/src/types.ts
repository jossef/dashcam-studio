export type Cam = 'F' | 'R' | 'C';
export const CAMS: Cam[] = ['F', 'R', 'C'];

export type Layout = 'focus' | 'grid' | 'solo';
export type Decoder = 'auto' | 'native' | 'compat';

/** [secondsIntoClip, lat, lon, kmh, headingDeg] */
export type TrackPoint = [number, number, number, number, number];

export interface GpsSummary {
  km: number;
  maxKmh: number;
  lat: number;
  lon: number;
}

export interface Clip {
  id: string;
  stem: string;
  category: string;
  start: string | null;
  seq: number;
  cams: Partial<Record<Cam, number>>; // cam -> bytes
  gps?: GpsSummary;
  /** parsed `start`, added on the client */
  date: Date | null;
}

export interface Category {
  name: string;
  count: number;
}

export interface Status {
  root: string | null;
  loading: boolean;
  error: string | null;
  ffmpeg: boolean;
  gpsFiles: string[];
  clipCount: number;
  platform: string;
  sep: string;
  /** none: nothing opened · connected · lost: card vanished · closed: user disconnected */
  connection: 'none' | 'connected' | 'lost' | 'closed';
  lastRoot: string | null;
  removable: boolean;
  label: string;
}

export interface CamInfo {
  name: string;
  file: string;
  size: number;
  mime?: string | null;
  codec?: string;
  width?: number;
  height?: number;
  fps?: number;
  duration?: number;
}

export interface ClipDetail {
  clip: Omit<Clip, 'date'>;
  cams: Partial<Record<Cam, CamInfo>>;
  track: TrackPoint[];
}

export interface Trip {
  id: string;
  category: string;
  /** chronological (oldest first) */
  clips: Clip[];
  start: Date | null;
  end: Date | null;
  km: number;
  maxKmh: number;
  bytes: number;
  day: string;
}

export interface DayGroup<T> {
  day: string;
  date: Date | null;
  items: T[];
}

export const FAVORITES_VIEW = '★';
