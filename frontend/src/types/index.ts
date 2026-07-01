// Canonical TypeScript types mirroring the proto schema.
// Will be replaced by generated types from buf once gen-proto.sh is run.

export type Side = 'friendly' | 'enemy' | 'neutral';

export type EntityType =
  | 'fixed_wing'
  | 'rotary_wing'
  | 'ground_vehicle'
  | 'dismounted_infantry'
  | 'surface_vessel'
  | 'submarine'
  | 'satellite'
  | 'uav'
  | 'missile'
  | 'radar_sensor'
  | 'base_fob';

export interface Position {
  lat: number;
  lon: number;
  alt_m: number;
}

export interface Entity {
  id: string;
  name: string;
  type: EntityType;
  side: Side;
  position: Position;
  attributes?: Record<string, string>;
}

export type MissionType =
  | 'cap' | 'cas' | 'strike' | 'patrol'
  | 'recon' | 'escort' | 'sar' | 'transit';

export type ROE = 'weapons_hold' | 'weapons_tight' | 'weapons_free';

export interface Waypoint {
  lat: number;
  lon: number;
  alt_m: number;
  speed_ms: number;
  hold_time_s: number;
}

export interface MissionTimeline {
  start_offset_s: number;
  expected_duration_s: number;
}

export interface EntityMission {
  entity_id: string;
  mission_type: MissionType;
  waypoints: Waypoint[];
  roe: ROE;
  objectives: string[];
  timeline: MissionTimeline;
}

export interface ScenarioSide {
  affiliation: Side;
  label: string;
}

export interface TerrainReference {
  min_lat: number;
  max_lat: number;
  min_lon: number;
  max_lon: number;
  description: string;
}

export interface Scenario {
  id: string;
  name: string;
  description: string;
  sides: ScenarioSide[];
  entities: Entity[];
  missions: EntityMission[];
  start_time: string;
  duration_s: number;
  terrain?: TerrainReference;
  engine_hint: string;
  created_at: string;
  updated_at: string;
}

export type RunStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface Run {
  id: string;
  scenario_id: string;
  engine_id: string;
  status: RunStatus;
  error?: string;
  batch_id?: string;
  created_at: string;
  updated_at: string;
}

export interface TrackPoint {
  timestamp_ms: number;
  lat: number;
  lon: number;
  alt_m: number;
  heading_deg: number;
  speed_ms: number;
  status: string;
}

export interface EntityTrack {
  entity_id: string;
  points: TrackPoint[];
}

export type EventType =
  | 'detection' | 'engagement' | 'kill' | 'damage'
  | 'launch' | 'waypoint_reached' | 'mission_complete';

export interface SimEvent {
  timestamp_ms: number;
  type: EventType;
  entity_id: string;
  target_entity_id?: string;
  detail?: string;
}

export interface KillChain {
  attacker_entity_id: string;
  target_entity_id: string;
  engaged_at_ms: number;
  killed_at_ms: number;
  weapon_ids?: string[];
}

export interface MOEMetric {
  key: string;
  value: number;
  unit?: string;
}

export interface SimResults {
  id: string;
  scenario_id: string;
  engine_id: string;
  run_id: string;
  entity_tracks: EntityTrack[];
  events: SimEvent[];
  kill_chains: KillChain[];
  moe_metrics: MOEMetric[];
  created_at: string;
}

// --- Batches (Monte Carlo replications) ---

export interface BatchMOEAggregate {
  key: string;
  unit: string;
  count: number;
  mean: number;
  stddev: number;
  min: number;
  max: number;
}

export interface BatchSummary {
  batch_id: string;
  scenario_id: string;
  engine_id: string;
  total: number;
  pending: number;
  running: number;
  completed: number;
  failed: number;
  runs: Run[];
  aggregated_moes: BatchMOEAggregate[];
}
