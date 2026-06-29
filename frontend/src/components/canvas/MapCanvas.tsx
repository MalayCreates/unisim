import {
  Viewer,
  Entity as CesiumEntity,
  PointGraphics,
  BillboardGraphics,
} from 'resium';
import * as Cesium from 'cesium';
import { useCallback } from 'react';
import { v4 as uuidv4 } from 'uuid';
import { useStore } from '../../store';
import type { Entity, EntityType } from '../../types';

// Map entity types to Cesium point colors for quick visual distinction.
const SIDE_COLORS: Record<string, Cesium.Color> = {
  friendly: Cesium.Color.CYAN,
  enemy:    Cesium.Color.RED,
  neutral:  Cesium.Color.YELLOW,
};

interface MapCanvasProps {
  defaultEntityType?: EntityType;
  defaultSide?: 'friendly' | 'enemy' | 'neutral';
}

export default function MapCanvas({
  defaultEntityType = 'fixed_wing',
  defaultSide = 'friendly',
}: MapCanvasProps) {
  const { activeScenario, upsertEntity, setSelectedEntityId, selectedEntityId } =
    useStore();

  const entities = activeScenario?.entities ?? [];

  const handleMapClick = useCallback(
    (movement: { position: Cesium.Cartesian2 }, viewer: Cesium.Viewer) => {
      const picked = viewer.scene.pickPosition(movement.position);
      if (!picked || !Cesium.defined(picked)) return;

      const cartographic = Cesium.Cartographic.fromCartesian(picked);
      const lat = Cesium.Math.toDegrees(cartographic.latitude);
      const lon = Cesium.Math.toDegrees(cartographic.longitude);
      const alt = cartographic.height > 0 ? cartographic.height : 0;

      const entity: Entity = {
        id: uuidv4(),
        name: `${defaultEntityType}-${Date.now()}`,
        type: defaultEntityType,
        side: defaultSide,
        position: { lat, lon, alt_m: alt },
        attributes: {},
      };
      upsertEntity(entity);
    },
    [defaultEntityType, defaultSide, upsertEntity],
  );

  return (
    <Viewer
      full
      timeline={false}
      animation={false}
      baseLayerPicker={false}
      onClick={(movement, viewer) => handleMapClick(movement as any, viewer)}
    >
      {entities.map((e) => (
        <CesiumEntity
          key={e.id}
          id={e.id}
          name={e.name}
          position={Cesium.Cartesian3.fromDegrees(
            e.position.lon,
            e.position.lat,
            e.position.alt_m,
          )}
          onClick={() => setSelectedEntityId(e.id)}
        >
          <PointGraphics
            pixelSize={selectedEntityId === e.id ? 14 : 10}
            color={SIDE_COLORS[e.side] ?? Cesium.Color.WHITE}
            outlineColor={Cesium.Color.BLACK}
            outlineWidth={1}
          />
        </CesiumEntity>
      ))}
    </Viewer>
  );
}
