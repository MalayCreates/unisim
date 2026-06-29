import type { EntityType, Side } from '../types';

// Domain groupings used by the layer filter controls.
export type Domain = 'air' | 'land' | 'sea' | 'space' | 'cyber';

export const DOMAINS: Domain[] = ['air', 'land', 'sea', 'space', 'cyber'];

export const ENTITY_DOMAIN: Record<EntityType, Domain> = {
  fixed_wing: 'air',
  rotary_wing: 'air',
  uav: 'air',
  missile: 'air',
  ground_vehicle: 'land',
  dismounted_infantry: 'land',
  base_fob: 'land',
  radar_sensor: 'land',
  surface_vessel: 'sea',
  submarine: 'sea',
  satellite: 'space',
};

export const DOMAIN_COLOR: Record<Domain, string> = {
  air: 'cyan',
  land: 'lime',
  sea: 'blue',
  space: 'violet',
  cyber: 'orange',
};

// Human-readable labels for entity types, ordered for the palette.
export const ENTITY_TYPES: { value: EntityType; label: string }[] = [
  { value: 'fixed_wing', label: 'Fixed-Wing' },
  { value: 'rotary_wing', label: 'Rotary-Wing' },
  { value: 'uav', label: 'UAV / Drone' },
  { value: 'missile', label: 'Missile' },
  { value: 'ground_vehicle', label: 'Ground Vehicle' },
  { value: 'dismounted_infantry', label: 'Infantry' },
  { value: 'surface_vessel', label: 'Surface Vessel' },
  { value: 'submarine', label: 'Submarine' },
  { value: 'satellite', label: 'Satellite' },
  { value: 'radar_sensor', label: 'Radar / Sensor' },
  { value: 'base_fob', label: 'Base / FOB' },
];

export const ENTITY_LABEL: Record<EntityType, string> = Object.fromEntries(
  ENTITY_TYPES.map((e) => [e.value, e.label]),
) as Record<EntityType, string>;

export const SIDES: { value: Side; label: string }[] = [
  { value: 'friendly', label: 'Friendly' },
  { value: 'enemy', label: 'Enemy' },
  { value: 'neutral', label: 'Neutral' },
];

// CSS color strings for each side (used for Cesium points and UI accents).
export const SIDE_CSS: Record<Side, string> = {
  friendly: '#22d3ee', // cyan
  enemy: '#ef4444', // red
  neutral: '#eab308', // yellow
};

export function domainForType(t: EntityType): Domain {
  return ENTITY_DOMAIN[t] ?? 'land';
}
