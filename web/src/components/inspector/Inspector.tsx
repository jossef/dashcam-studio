import { useApp } from '../../store/app';
import { FilesPanel } from './FilesPanel';
import { MapView } from './MapView';
import { Telemetry } from './Telemetry';

export function Inspector() {
  const grid = useApp((s) => s.layout === 'grid');
  return (
    <aside className="inspector" aria-label="מפה ופרטים">
      {/* in grid layout the map lives on the stage instead */}
      {!grid && (
        <div className="map-home">
          <MapView />
        </div>
      )}
      <Telemetry />
      <FilesPanel />
    </aside>
  );
}
