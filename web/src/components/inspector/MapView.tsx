import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { useEffect, useRef, useState } from 'react';
import { t } from '../../i18n/he';
import { api } from '../../lib/api';
import { fmtClock } from '../../lib/format';
import { gpsAt, nearestIndex } from '../../lib/gps';
import { usePlayhead } from '../../hooks/useEngine';
import { engine } from '../../player/engine';
import { openClip } from '../../store/actions';
import { useApp } from '../../store/app';
import type { TrackPoint } from '../../types';

const tripTrackCache = new Map<string, Record<string, TrackPoint[]>>();

interface MapParts {
  map: L.Map;
  clipLine: L.Polyline;
  glow: L.Polyline;
  tripLayer: L.LayerGroup;
  car: L.Marker;
  followPausedUntil: number;
}

/**
 * Route map: the current clip in amber, the rest of the drive faded (click it to
 * jump there), and a car marker that follows the playhead.
 */
export function MapView() {
  const host = useRef<HTMLDivElement>(null);
  const parts = useRef<MapParts | null>(null);
  const [tilesFailed, setTilesFailed] = useState(false);
  const current = useApp((s) => s.current);
  const trip = useApp((s) => (s.current ? s.tripOf.get(s.current.id) : undefined));
  const track = current?.detail?.track ?? [];

  // ------------------------------------------------------------ create once
  useEffect(() => {
    const map = L.map(host.current!, { zoomControl: true, attributionControl: true }).setView([32.07, 34.82], 13);
    const tiles = L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    }).addTo(map);
    let ok = 0;
    tiles.on('tileload', () => (ok++, setTilesFailed(false)));
    tiles.on('tileerror', () => !ok && setTilesFailed(true));
    const tripLayer = L.layerGroup().addTo(map);
    const glow = L.polyline([], { color: '#ffb238', weight: 12, opacity: 0.18, interactive: false }).addTo(map);
    const clipLine = L.polyline([], { color: '#ffb238', weight: 4, opacity: 0.95 }).addTo(map);
    const car = L.marker([0, 0], {
      icon: L.divIcon({ className: 'car-marker', html: '<div class="car-icon"><i></i></div>', iconSize: [22, 22], iconAnchor: [11, 11] }),
      interactive: false,
      keyboard: false,
    });
    const p: MapParts = { map, clipLine, glow, tripLayer, car, followPausedUntil: 0 };
    clipLine.on('click', (e: L.LeafletMouseEvent) => {
      const tr = useApp.getState().current?.detail?.track;
      if (tr?.length) engine.seek(tr[nearestIndex(tr, e.latlng.lat, e.latlng.lng)][0]);
    });
    const pauseFollow = () => (p.followPausedUntil = performance.now() + 8000);
    map.on('dragstart', pauseFollow);
    map.getContainer().addEventListener('wheel', pauseFollow, { passive: true });
    parts.current = p;
    const ro = new ResizeObserver(() => map.invalidateSize());
    ro.observe(host.current!);
    return () => {
      ro.disconnect();
      map.remove();
      parts.current = null;
    };
  }, []);

  // ------------------------------------------------------------ clip route
  useEffect(() => {
    const p = parts.current;
    if (!p) return;
    const latlngs = track.map((q) => [q[1], q[2]] as L.LatLngTuple);
    p.clipLine.setLatLngs(latlngs);
    p.glow.setLatLngs(latlngs);
    if (latlngs.length) {
      p.car.addTo(p.map);
      p.map.fitBounds(L.latLngBounds(latlngs).pad(0.35), { maxZoom: 17, animate: true });
    } else p.car.remove();
  }, [track]);

  // ------------------------------------------------------------ rest of the drive
  useEffect(() => {
    const p = parts.current;
    if (!p || !current) return;
    p.tripLayer.clearLayers();
    const ids = trip?.clips.filter((c) => c.gps).map((c) => c.id) ?? [];
    if (!trip || !ids.length) return;
    let cancelled = false;
    const draw = (tracks: Record<string, TrackPoint[]>) => {
      if (cancelled) return;
      p.tripLayer.clearLayers();
      const { byId } = useApp.getState();
      for (const [cid, tr] of Object.entries(tracks)) {
        if (cid === current.id || !tr.length) continue;
        const line = L.polyline(
          tr.map((q) => [q[1], q[2]] as L.LatLngTuple),
          { color: '#8a7a5c', weight: 3, opacity: 0.55 },
        );
        line.on('click', (e: L.LeafletMouseEvent) =>
          openClip(cid, { t: Math.max(0, tr[nearestIndex(tr, e.latlng.lat, e.latlng.lng)][0]), play: engine.playing }),
        );
        const c = byId.get(cid);
        if (c?.date) line.bindTooltip(fmtClock(c.date), { sticky: true, direction: 'top', opacity: 0.9 });
        p.tripLayer.addLayer(line);
      }
    };
    const cached = tripTrackCache.get(trip.id);
    if (cached) draw(cached);
    else
      api
        .tracks(ids)
        .then((r) => {
          tripTrackCache.set(trip.id, r.tracks);
          draw(r.tracks);
        })
        .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [trip, current?.id]);

  return (
    <div className="map-wrap">
      <div ref={host} className="map" dir="ltr" />
      <CarFollower parts={parts} track={track} />
      {(!track.length || tilesFailed) && <div className="map-empty">{!track.length ? t.noGps : t.mapOffline}</div>}
    </div>
  );
}

/** Isolated so only this tiny component re-renders at playhead rate. */
function CarFollower({ parts, track }: { parts: React.RefObject<MapParts | null>; track: TrackPoint[] }) {
  const now = usePlayhead(10);
  useEffect(() => {
    const p = parts.current;
    const g = gpsAt(track, now);
    if (!p || !g) return;
    p.car.setLatLng([g.lat, g.lon]);
    const icon = p.car.getElement()?.firstElementChild as HTMLElement | null;
    if (icon) icon.style.transform = `rotate(${g.hdg}deg)`;
    if (engine.playing && performance.now() > p.followPausedUntil) {
      const b = p.map.getBounds().pad(-0.2);
      if (!b.contains([g.lat, g.lon])) p.map.panTo([g.lat, g.lon], { animate: true, duration: 0.6 });
    }
  }, [now, track, parts]);
  return null;
}
