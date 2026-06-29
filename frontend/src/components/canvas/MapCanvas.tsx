import { useEffect, useMemo } from 'react';
import {
  Viewer,
  Entity,
  PointGraphics,
  PolylineGraphics,
  LabelGraphics,
  ImageryLayer,
  useCesium,
} from 'resium';
import * as Cesium from 'cesium';
import { useStore } from '../../store';
import { SIDE_CSS, domainForType } from '../../lib/domain';
import { interpolateTrack, resultsTimeRange } from '../../lib/playback';

const KILLED_CSS = '#6b7280'; // gray for killed entities during playback

// MapInteractions wires the left-click handler once the Cesium viewer exists.
// Using useCesium() (rather than a ref) guarantees the viewer is mounted.
function MapInteractions() {
  const { viewer } = useCesium();
  useEffect(() => {
    if (!viewer) return;
    const handler = new Cesium.ScreenSpaceEventHandler(viewer.scene.canvas);
    handler.setInputAction((e: Cesium.ScreenSpaceEventHandler.PositionedEvent) => {
      handleClick(viewer, e.position);
    }, Cesium.ScreenSpaceEventType.LEFT_CLICK);
    return () => handler.destroy();
  }, [viewer]);
  return null;
}

export default function MapCanvas() {
  const activeScenario = useStore((s) => s.activeScenario);
  const results = useStore((s) => s.results);
  const visibleDomains = useStore((s) => s.visibleDomains);
  const visibleEngines = useStore((s) => s.visibleEngines);
  const selectedEntityId = useStore((s) => s.selectedEntityId);
  const playbackTimeMS = useStore((s) => s.playbackTimeMS);

  // Token-free OSM imagery, created once.
  const osm = useMemo(
    () => new Cesium.OpenStreetMapImageryProvider({ url: 'https://tile.openstreetmap.org/' }),
    [],
  );

  const timeRange = useMemo(() => (results ? resultsTimeRange(results) : null), [results]);
  const inPlayback = !!results && !!timeRange;

  const entityMeta = useMemo(() => {
    const m = new Map<string, { type: string; side: string }>();
    for (const e of activeScenario?.entities ?? []) {
      m.set(e.id, { type: e.type, side: e.side });
    }
    return m;
  }, [activeScenario]);

  const selectedMission = useMemo(
    () => (activeScenario?.missions ?? []).find((mi) => mi.entity_id === selectedEntityId),
    [activeScenario, selectedEntityId],
  );

  return (
    <Viewer
      style={{ width: '100%', height: '100%' }}
      baseLayer={false}
      baseLayerPicker={false}
      geocoder={false}
      timeline={false}
      animation={false}
      infoBox={false}
      selectionIndicator={false}
      fullscreenButton={false}
      homeButton={false}
      navigationHelpButton={false}
      sceneModePicker={false}
    >
      <ImageryLayer imageryProvider={osm} />
      <MapInteractions />

      {/* --- Playback rendering: entities at interpolated track positions --- */}
      {inPlayback &&
        results.entity_tracks.map((track) => {
          const meta = entityMeta.get(track.entity_id);
          if (meta && !visibleDomains.includes(domainForType(meta.type as never))) return null;
          if (!visibleEngines.includes(results.engine_id)) return null;
          const p = interpolateTrack(track, timeRange.startMs + playbackTimeMS);
          if (!p) return null;
          const css = p.status === 'killed' ? KILLED_CSS : SIDE_CSS[(meta?.side as never) ?? 'neutral'];
          return (
            <Entity
              key={track.entity_id}
              id={track.entity_id}
              position={Cesium.Cartesian3.fromDegrees(p.lon, p.lat, p.alt_m)}
            >
              <PointGraphics
                pixelSize={selectedEntityId === track.entity_id ? 16 : 11}
                color={Cesium.Color.fromCssColorString(css)}
                outlineColor={Cesium.Color.BLACK}
                outlineWidth={1}
              />
            </Entity>
          );
        })}

      {/* --- Edit rendering: static entities + selected waypoints --- */}
      {!inPlayback &&
        (activeScenario?.entities ?? []).map((e) => {
          if (!visibleDomains.includes(domainForType(e.type))) return null;
          const selected = selectedEntityId === e.id;
          return (
            <Entity
              key={e.id}
              id={e.id}
              position={Cesium.Cartesian3.fromDegrees(e.position.lon, e.position.lat, e.position.alt_m)}
            >
              <PointGraphics
                pixelSize={selected ? 16 : 11}
                color={Cesium.Color.fromCssColorString(SIDE_CSS[e.side])}
                outlineColor={selected ? Cesium.Color.WHITE : Cesium.Color.BLACK}
                outlineWidth={selected ? 2 : 1}
              />
              {selected && (
                <LabelGraphics
                  text={e.name}
                  font="12px monospace"
                  pixelOffset={new Cesium.Cartesian2(0, -18)}
                  fillColor={Cesium.Color.WHITE}
                  showBackground
                  backgroundColor={new Cesium.Color(0, 0, 0, 0.6)}
                />
              )}
            </Entity>
          );
        })}

      {/* --- Selected entity's waypoint path (edit mode only) --- */}
      {!inPlayback && selectedMission && selectedMission.waypoints.length > 0 && (() => {
        const start = activeScenario?.entities.find((e) => e.id === selectedMission.entity_id);
        const coords: number[] = [];
        if (start) coords.push(start.position.lon, start.position.lat, start.position.alt_m);
        for (const w of selectedMission.waypoints) coords.push(w.lon, w.lat, w.alt_m);
        return (
          <>
            <Entity id={`wp-line-${selectedMission.entity_id}`}>
              <PolylineGraphics
                positions={Cesium.Cartesian3.fromDegreesArrayHeights(coords)}
                width={2}
                material={Cesium.Color.fromCssColorString('#fbbf24').withAlpha(0.8)}
                clampToGround={false}
              />
            </Entity>
            {selectedMission.waypoints.map((w, i) => (
              <Entity
                key={`wp-${selectedMission.entity_id}-${i}`}
                id={`wp-${selectedMission.entity_id}-${i}`}
                position={Cesium.Cartesian3.fromDegrees(w.lon, w.lat, w.alt_m)}
              >
                <PointGraphics pixelSize={8} color={Cesium.Color.fromCssColorString('#fbbf24')} />
                <LabelGraphics
                  text={String(i + 1)}
                  font="11px monospace"
                  pixelOffset={new Cesium.Cartesian2(0, -14)}
                  fillColor={Cesium.Color.fromCssColorString('#fbbf24')}
                />
              </Entity>
            ))}
          </>
        );
      })()}
    </Viewer>
  );
}

// handleClick reads fresh store state so the listener can be attached once.
function handleClick(viewer: Cesium.Viewer, windowPos: Cesium.Cartesian2) {
  const state = useStore.getState();
  const { interactionMode, activeScenario, selectedEntityId } = state;
  if (!activeScenario) return;

  // Entity hit-test first: clicking one of our entities selects it.
  const picked = viewer.scene.pick(windowPos);
  if (picked?.id?.id && activeScenario.entities.some((e) => e.id === picked.id.id)) {
    state.selectEntity(picked.id.id);
    return;
  }

  // Otherwise resolve a ground position on the ellipsoid.
  const cartesian = viewer.camera.pickEllipsoid(windowPos, viewer.scene.globe.ellipsoid);
  if (!cartesian) return;
  const carto = Cesium.Cartographic.fromCartesian(cartesian);
  const lat = Cesium.Math.toDegrees(carto.latitude);
  const lon = Cesium.Math.toDegrees(carto.longitude);

  if (interactionMode === 'place') {
    state.addEntityAt(lat, lon);
  } else if (interactionMode === 'waypoint' && selectedEntityId) {
    state.addWaypointAt(selectedEntityId, lat, lon);
  } else {
    state.selectEntity(null);
  }
}
