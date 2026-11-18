import type { SVGProps } from 'react';

const S = { fill: 'none', stroke: 'currentColor', strokeWidth: 1.8, strokeLinecap: 'round', strokeLinejoin: 'round' } as const;

const paths = {
  play: <path d="M7 4.5v15l12.5-7.5z" fill="currentColor" />,
  pause: <path d="M6.5 4.5h4v15h-4zM13.5 4.5h4v15h-4z" fill="currentColor" />,
  prev: <path d="M6 5h2v14H6zM19 5v14L9 12z" fill="currentColor" />,
  next: <path d="M16 5h2v14h-2zM5 5v14l10-7z" fill="currentColor" />,
  back: (
    <>
      <path d="M12 5V2L7 6l5 4V7a6 6 0 1 1-6 6H4a8 8 0 1 0 8-8z" fill="currentColor" />
      <text x="12" y="16.4" fontSize="6.5" textAnchor="middle" fill="currentColor" fontFamily="sans-serif" fontWeight="700">10</text>
    </>
  ),
  fwd: (
    <>
      <path d="M12 5V2l5 4-5 4V7a6 6 0 1 0 6 6h2a8 8 0 1 1-8-8z" fill="currentColor" />
      <text x="12" y="16.4" fontSize="6.5" textAnchor="middle" fill="currentColor" fontFamily="sans-serif" fontWeight="700">10</text>
    </>
  ),
  vol: (
    <>
      <path d="M4 9h4l5-4v14l-5-4H4z" fill="currentColor" />
      <path d="M16 8.5a5 5 0 0 1 0 7M18.5 6a8.5 8.5 0 0 1 0 12" {...S} />
    </>
  ),
  mute: (
    <>
      <path d="M4 9h4l5-4v14l-5-4H4z" fill="currentColor" />
      <path d="M16.5 9.5l5 5M21.5 9.5l-5 5" {...S} />
    </>
  ),
  focus: (
    <>
      <rect x="8.5" y="4.5" width="13" height="15" rx="1.5" fill="currentColor" />
      <rect x="2.5" y="4.5" width="4.5" height="7" rx="1" fill="currentColor" opacity=".55" />
      <rect x="2.5" y="12.5" width="4.5" height="7" rx="1" fill="currentColor" opacity=".55" />
    </>
  ),
  grid: (
    <>
      <rect x="2.5" y="4" width="9" height="7.5" rx="1.2" fill="currentColor" />
      <rect x="12.5" y="4" width="9" height="7.5" rx="1.2" fill="currentColor" />
      <rect x="12.5" y="12.5" width="9" height="7.5" rx="1.2" fill="currentColor" />
      <rect x="2.5" y="12.5" width="9" height="7.5" rx="1.2" fill="currentColor" opacity=".55" />
    </>
  ),
  solo: <rect x="2.5" y="4.5" width="19" height="15" rx="1.5" fill="currentColor" />,
  camera: (
    <>
      <path d="M8.5 5l-1.5 2H4a1.5 1.5 0 0 0-1.5 1.5v10A1.5 1.5 0 0 0 4 20h16a1.5 1.5 0 0 0 1.5-1.5v-10A1.5 1.5 0 0 0 20 7h-3l-1.5-2z" {...S} />
      <circle cx="12" cy="13" r="3.6" {...S} />
    </>
  ),
  full: <path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5" {...S} strokeWidth={2} />,
  download: <path d="M12 3.5v11m0 0l-4.5-4.5M12 14.5l4.5-4.5M4.5 17.5v2h15v-2" {...S} strokeWidth={2} />,
  zip: (
    <>
      <path d="M6 2.8h8.5L19 7.3v13.9H6z" {...S} />
      <path d="M10.5 4.5h2M10.5 7h2M10.5 9.5h2M10.5 12h2" {...S} />
      <rect x="9.8" y="14" width="3.4" height="4" rx=".8" fill="currentColor" />
    </>
  ),
  map: (
    <>
      <path d="M9 4L3.5 6v14L9 18l6 2 5.5-2V4L15 6z" {...S} />
      <path d="M9 4v14M15 6v14" {...S} />
    </>
  ),
  list: <path d="M4 6h16M4 12h16M10 18h10" {...S} strokeWidth={2} />,
  folder: <path d="M3 6.5A1.5 1.5 0 0 1 4.5 5h4.2l2 2.2h8.8A1.5 1.5 0 0 1 21 8.7v9.8a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 18.5z" {...S} />,
  drive: (
    <>
      <rect x="6" y="2.5" width="12" height="19" rx="2" {...S} />
      <path d="M9 6h6M9 9h6" {...S} />
      <circle cx="12" cy="16.5" r="1.4" fill="currentColor" />
    </>
  ),
  up: <path d="M12 19V5m0 0l-6 6m6-6l6 6" {...S} strokeWidth={2} />,
  chev: <path d="M15 6l-6 6 6 6" {...S} strokeWidth={2} />,
  keys: (
    <>
      <rect x="2.5" y="6" width="19" height="12" rx="2" {...S} />
      <path d="M6 10h1M9.5 10h1M13 10h1M16.5 10h1.5M7 14h10" {...S} />
    </>
  ),
  refresh: <path d="M20 12a8 8 0 1 1-2.3-5.6M20 4v4.5h-4.5" {...S} strokeWidth={2} />,
  x: <path d="M6 6l12 12M18 6L6 18" {...S} strokeWidth={2} />,
  pin: (
    <>
      <path d="M12 21s-6.5-6.1-6.5-11a6.5 6.5 0 0 1 13 0c0 4.9-6.5 11-6.5 11z" {...S} />
      <circle cx="12" cy="10" r="2.3" fill="currentColor" />
    </>
  ),
  copy: (
    <>
      <rect x="8" y="8" width="12" height="12" rx="2" {...S} />
      <path d="M16 8V5.5A1.5 1.5 0 0 0 14.5 4h-9A1.5 1.5 0 0 0 4 5.5v9A1.5 1.5 0 0 0 5.5 16H8" {...S} />
    </>
  ),
  star: <path d="M12 3.2l2.7 5.6 6.1.8-4.5 4.2 1.1 6.1L12 17l-5.4 2.9 1.1-6.1-4.5-4.2 6.1-.8z" {...S} />,
  starFill: <path d="M12 3.2l2.7 5.6 6.1.8-4.5 4.2 1.1 6.1L12 17l-5.4 2.9 1.1-6.1-4.5-4.2 6.1-.8z" fill="currentColor" stroke="currentColor" strokeWidth={1.4} strokeLinejoin="round" />,
  check: <path d="M5 12.5l4.5 4.5L19 7.5" {...S} strokeWidth={2.6} />,
  minus: <path d="M6 12h12" {...S} strokeWidth={2.6} />,
  eject: (
    <>
      <path d="M12 4.5L4.5 14h15z" fill="currentColor" />
      <rect x="4.5" y="16.5" width="15" height="3" rx="1" fill="currentColor" />
    </>
  ),
  card: (
    <>
      <path d="M8 2.5h9.5A1.5 1.5 0 0 1 19 4v16a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 5 20V5.5z" {...S} />
      <path d="M9 3v4M12 3v4M15 3v4" {...S} />
    </>
  ),
  warn: (
    <>
      <path d="M12 3.5L2.5 20h19z" {...S} />
      <path d="M12 10v4.5" {...S} strokeWidth={2} />
      <circle cx="12" cy="17.2" r="1.1" fill="currentColor" />
    </>
  ),
} as const;

export type IconName = keyof typeof paths;

export function Icon({ name, size, ...rest }: { name: IconName; size?: number } & SVGProps<SVGSVGElement>) {
  return (
    <svg className="icon" viewBox="0 0 24 24" width={size} height={size} aria-hidden="true" {...rest}>
      {paths[name]}
    </svg>
  );
}
