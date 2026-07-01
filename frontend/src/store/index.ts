import { create } from 'zustand';
import { v4 as uuidv4 } from 'uuid';
import type {
  BatchSummary,
  Entity,
  EntityMission,
  EntityType,
  RunStatus,
  Scenario,
  Side,
  SimResults,
  Waypoint,
} from '../types';
import { DOMAINS, type Domain } from '../lib/domain';

export type InteractionMode = 'select' | 'place' | 'waypoint';

function defaultMission(entityId: string): EntityMission {
  return {
    entity_id: entityId,
    mission_type: 'patrol',
    waypoints: [],
    roe: 'weapons_tight',
    objectives: [],
    timeline: { start_offset_s: 0, expected_duration_s: 0 },
  };
}

interface AppStore {
  // --- Scenarios ---
  scenarios: Scenario[];
  activeScenario: Scenario | null;
  dirty: boolean;
  setScenarios: (s: Scenario[]) => void;
  upsertScenario: (s: Scenario) => void;
  removeScenario: (id: string) => void;
  setActiveScenario: (s: Scenario | null) => void;
  markClean: () => void;

  // --- Interaction ---
  interactionMode: InteractionMode;
  paletteType: EntityType;
  paletteSide: Side;
  selectedEntityId: string | null;
  setInteractionMode: (m: InteractionMode) => void;
  setPalette: (type: EntityType, side: Side) => void;
  selectEntity: (id: string | null) => void;

  // --- Entity editing (operate on activeScenario) ---
  addEntityAt: (lat: number, lon: number, altM?: number) => void;
  updateEntity: (e: Entity) => void;
  removeEntity: (id: string) => void;

  // --- Mission editing ---
  missionFor: (entityId: string) => EntityMission | undefined;
  updateMission: (m: EntityMission) => void;
  addWaypointAt: (entityId: string, lat: number, lon: number, altM?: number) => void;
  removeWaypoint: (entityId: string, index: number) => void;

  // --- Runs / results ---
  runStatus: RunStatus | null;
  activeRunId: string | null;
  results: SimResults | null;
  setRunStatus: (s: RunStatus | null) => void;
  setActiveRunId: (id: string | null) => void;
  setResults: (r: SimResults | null) => void;

  // --- Batches (Monte Carlo replications) ---
  batchResult: BatchSummary | null;
  setBatchResult: (b: BatchSummary | null) => void;

  // --- Playback ---
  playing: boolean;
  playbackTimeMS: number; // relative ms from track start
  setPlaying: (p: boolean) => void;
  setPlaybackTimeMS: (t: number) => void;

  // --- Layer filters ---
  visibleDomains: Domain[];
  visibleEngines: string[];
  toggleDomain: (d: Domain) => void;
  setVisibleDomains: (d: Domain[]) => void;
  toggleEngine: (id: string) => void;
  setVisibleEngines: (ids: string[]) => void;
}

// Helper: immutably patch the active scenario and flag it dirty.
function patchActive(
  scenario: Scenario | null,
  fn: (s: Scenario) => Scenario,
): Partial<AppStore> {
  if (!scenario) return {};
  return { activeScenario: fn(scenario), dirty: true };
}

export const useStore = create<AppStore>((set, get) => ({
  // --- Scenarios ---
  scenarios: [],
  activeScenario: null,
  dirty: false,
  setScenarios: (scenarios) => set({ scenarios }),
  upsertScenario: (sc) =>
    set((s) => ({
      scenarios: s.scenarios.some((x) => x.id === sc.id)
        ? s.scenarios.map((x) => (x.id === sc.id ? sc : x))
        : [...s.scenarios, sc],
    })),
  removeScenario: (id) =>
    set((s) => ({ scenarios: s.scenarios.filter((x) => x.id !== id) })),
  setActiveScenario: (sc) =>
    set({
      activeScenario: sc,
      dirty: false,
      selectedEntityId: null,
      results: null,
      batchResult: null,
      runStatus: null,
      activeRunId: null,
      playing: false,
      playbackTimeMS: 0,
      interactionMode: 'select',
    }),
  markClean: () => set({ dirty: false }),

  // --- Interaction ---
  interactionMode: 'select',
  paletteType: 'fixed_wing',
  paletteSide: 'friendly',
  selectedEntityId: null,
  setInteractionMode: (interactionMode) => set({ interactionMode }),
  setPalette: (paletteType, paletteSide) => set({ paletteType, paletteSide }),
  selectEntity: (selectedEntityId) => set({ selectedEntityId }),

  // --- Entity editing ---
  addEntityAt: (lat, lon, altM = 0) =>
    set((s) => {
      if (!s.activeScenario) return {};
      const entity: Entity = {
        id: uuidv4(),
        name: `${s.paletteType}-${s.activeScenario.entities.length + 1}`,
        type: s.paletteType,
        side: s.paletteSide,
        position: { lat, lon, alt_m: altM },
        attributes: {},
      };
      return {
        ...patchActive(s.activeScenario, (sc) => ({
          ...sc,
          entities: [...sc.entities, entity],
        })),
        selectedEntityId: entity.id,
      };
    }),
  updateEntity: (e) =>
    set((s) =>
      patchActive(s.activeScenario, (sc) => ({
        ...sc,
        entities: sc.entities.map((x) => (x.id === e.id ? e : x)),
      })),
    ),
  removeEntity: (id) =>
    set((s) => ({
      ...patchActive(s.activeScenario, (sc) => ({
        ...sc,
        entities: sc.entities.filter((x) => x.id !== id),
        missions: (sc.missions ?? []).filter((m) => m.entity_id !== id),
      })),
      selectedEntityId: s.selectedEntityId === id ? null : s.selectedEntityId,
    })),

  // --- Mission editing ---
  missionFor: (entityId) =>
    (get().activeScenario?.missions ?? []).find((m) => m.entity_id === entityId),
  updateMission: (m) =>
    set((s) =>
      patchActive(s.activeScenario, (sc) => {
        const missions = sc.missions ?? [];
        return {
          ...sc,
          missions: missions.some((x) => x.entity_id === m.entity_id)
            ? missions.map((x) => (x.entity_id === m.entity_id ? m : x))
            : [...missions, m],
        };
      }),
    ),
  addWaypointAt: (entityId, lat, lon, altM = 0) =>
    set((s) =>
      patchActive(s.activeScenario, (sc) => {
        const missions = sc.missions ?? [];
        const existing = missions.find((m) => m.entity_id === entityId);
        const wp: Waypoint = { lat, lon, alt_m: altM, speed_ms: 200, hold_time_s: 0 };
        if (existing) {
          return {
            ...sc,
            missions: missions.map((m) =>
              m.entity_id === entityId ? { ...m, waypoints: [...m.waypoints, wp] } : m,
            ),
          };
        }
        const created = defaultMission(entityId);
        created.waypoints = [wp];
        return { ...sc, missions: [...missions, created] };
      }),
    ),
  removeWaypoint: (entityId, index) =>
    set((s) =>
      patchActive(s.activeScenario, (sc) => ({
        ...sc,
        missions: (sc.missions ?? []).map((m) =>
          m.entity_id === entityId
            ? { ...m, waypoints: m.waypoints.filter((_, i) => i !== index) }
            : m,
        ),
      })),
    ),

  // --- Runs / results ---
  runStatus: null,
  activeRunId: null,
  results: null,
  setRunStatus: (runStatus) => set({ runStatus }),
  setActiveRunId: (activeRunId) => set({ activeRunId }),
  setResults: (results) =>
    set({
      results,
      batchResult: null,
      playbackTimeMS: 0,
      playing: false,
      visibleEngines: results ? [results.engine_id] : [],
    }),

  // --- Batches (Monte Carlo replications) ---
  batchResult: null,
  setBatchResult: (batchResult) => set({ batchResult, results: null }),

  // --- Playback ---
  playing: false,
  playbackTimeMS: 0,
  setPlaying: (playing) => set({ playing }),
  setPlaybackTimeMS: (playbackTimeMS) => set({ playbackTimeMS }),

  // --- Layer filters ---
  visibleDomains: [...DOMAINS],
  visibleEngines: [],
  toggleDomain: (d) =>
    set((s) => ({
      visibleDomains: s.visibleDomains.includes(d)
        ? s.visibleDomains.filter((x) => x !== d)
        : [...s.visibleDomains, d],
    })),
  setVisibleDomains: (visibleDomains) => set({ visibleDomains }),
  toggleEngine: (id) =>
    set((s) => ({
      visibleEngines: s.visibleEngines.includes(id)
        ? s.visibleEngines.filter((x) => x !== id)
        : [...s.visibleEngines, id],
    })),
  setVisibleEngines: (visibleEngines) => set({ visibleEngines }),
}));
