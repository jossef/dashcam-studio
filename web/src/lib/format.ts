export const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));
export const pad = (n: number, w = 2) => String(Math.floor(n)).padStart(w, '0');
export const sum = <T,>(a: T[], f: (x: T) => number | undefined | null) => a.reduce((s, x) => s + (f(x) || 0), 0);

export const fmtClock = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
export const fmtHM = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}`;
export const fmtDur = (s: number) => {
  s = Math.max(0, Math.floor(s || 0));
  return s >= 3600 ? `${Math.floor(s / 3600)}:${pad((s / 60) % 60)}:${pad(s % 60)}` : `${Math.floor(s / 60)}:${pad(s % 60)}`;
};
/** Sizes are wrapped in a left-to-right isolate so "916 MB" doesn't flip inside Hebrew text. */
export const fmtBytes = (b: number) => `⁦${bytes(b)}⁩`;
const bytes = (b: number) => {
  if (!b) return '0 B';
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(u.length - 1, Math.floor(Math.log(b) / Math.log(1024)));
  return `${(b / 1024 ** i).toFixed(i >= 3 ? 2 : 0)} ${u[i]}`;
};
export const fmtKm = (km?: number | null) => (km == null ? '—' : km < 10 ? km.toFixed(1) : Math.round(km).toString());
export const dayKey = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

const he = (o: Intl.DateTimeFormatOptions) => new Intl.DateTimeFormat('he-IL', o);
const WEEKDAY = he({ weekday: 'short' });
const WEEKDAY_LONG = he({ weekday: 'long' });
const MONTH = he({ month: 'short' });
const DAY_MONTH = he({ day: 'numeric', month: 'long' });
const FULL = he({ weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });

/** "יום ב׳" → "ב׳" for compact chips */
export const weekdayShort = (d: Date) => WEEKDAY.format(d).replace(/^יום\s*/, '');
export const monthShort = (d: Date) => MONTH.format(d);
export const dayTitle = (d: Date) => `${WEEKDAY_LONG.format(d)}, ${DAY_MONTH.format(d)}`;
export const fullDate = (d: Date) => FULL.format(d);

export const COMPASS = ['צפון', 'צפון-מזרח', 'מזרח', 'דרום-מזרח', 'דרום', 'דרום-מערב', 'מערב', 'צפון-מערב'];
export const compass = (deg: number) => COMPASS[Math.round(deg / 45) % 8];

export const clipBytes = (c: { cams: Partial<Record<string, number>> }) => sum(Object.values(c.cams), (x) => x);

const SHORT_DATE = he({ day: 'numeric', month: 'short', year: 'numeric' });
/** "9 בנוב׳ 2026" or "9 בנוב׳ 2026 – 13 בנוב׳ 2026" */
export const fmtDateRange = (dates: Date[]) => {
  const ts = dates.map((d) => d.getTime());
  const a = new Date(Math.min(...ts));
  const b = new Date(Math.max(...ts));
  return dayKey(a) === dayKey(b) ? SHORT_DATE.format(a) : `${SHORT_DATE.format(a)} – ${SHORT_DATE.format(b)}`;
};
