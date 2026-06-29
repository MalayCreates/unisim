import { create } from 'zustand';
import type { Entity, Run, Scenario, SimResults } from '../types';

interface ScenarioState {
  // Scenario list
  scenarios: Scenario[];
  setScenarios: (s: Scenario[]) => void;
  upsertScenario: (s: Scenario) => void;
  removeScenario: (id: string) => void;

  // Active scenario being edited
  activeScenario: Scenario | null;
  setActiveScenario: (s: Scenario | null) => void;

  // Selected entity on the map
  selectedEntityId: string | null;
  setSelectedEntityId: (id: string | null) => void;

  // Convenience: mutate entities within the active scenario
  upsertEntity: (e: Entity) => void;
  removeEntity: (id: string) => void;
}

interface RunState {
  runs: Run[];
  setRuns: (runs: Run[]) => void;
  upsertRun: (r: Run) => void;

  activeRunId: string | null;
  setActiveRunId: (id: string | null) => void;

  results: SimResults | null;
  setResults: (r: SimResults | null) => void;

  // Playback scrub position (ms into the scenario)
  playbackTimeMS: number;
  setPlaybackTimeMS: (t: number) => void;
}

interface LayerState {
  // Which domains are visible
  visibleDomains: Set<string>;
  toggleDomain: (domain: string) => void;

  // Which engine IDs are visible in results
  visibleEngines: Set<string>;
  toggleEngine: (engineId: string) => void;
}

type AppStore = ScenarioState & RunState & LayerState;

export const useStore = create<AppStore>((set) => ({
  // --- ScenarioState ---
  scenarios: [],
  setScenarios: (scenarios) => set({ scenarios }),
  upsertScenario: (sc) =>
    set((s) => ({
      scenarios: s.scenarios.some((x) => x.id === sc.id)
        ? s.scenarios.map((x) => (x.id === sc.id ? sc : x))
        : [...s.scenarios, sc],
    })),
  removeScenario: (id) =>
    set((s) => ({ scenarios: s.scenarios.filter((x) => x.id !== id) })),

  activeScenario: null,
  setActiveScenario: (sc) => set({ activeScenario: sc }),

  selectedEntityId: null,
  setSelectedEntityId: (id) => set({ selectedEntityId: id }),

  upsertEntity: (entity) =>
    set((s) => {
      if (!s.activeScenario) return {};
      const entities = s.activeScenario.entities ?? [];
      const updated = entities.some((e) => e.id === entity.id)
        ? entities.map((e) => (e.id === entity.id ? entity : e))
        : [...entities, entity];
      return { activeScenario: { ...s.activeScenario, entities: updated } };
    }),

  removeEntity: (id) =>
    set((s) => {
      if (!s.activeScenario) return {};
      return {
        activeScenario: {
          ...s.activeScenario,
          entities: s.activeScenario.entities.filter((e) => e.id !== id),
        },
      };
    }),

  // --- RunState ---
  runs: [],
  setRuns: (runs) => set({ runs }),
  upsertRun: (r) =>
    set((s) => ({
      runs: s.runs.some((x) => x.id === r.id)
        ? s.runs.map((x) => (x.id === r.id ? r : x))
        : [...s.runs, r],
    })),

  activeRunId: null,
  setActiveRunId: (id) => set({ activeRunId: id }),

  results: null,
  setResults: (results) => set({ results }),

  playbackTimeMS: 0,
  setPlaybackTimeMS: (t) => set({ playbackTimeMS: t }),

  // --- LayerState ---
  visibleDomains: new Set(['air', 'land', 'sea', 'space', 'cyber']),
  toggleDomain: (domain) =>
    set((s) => {
      const next = new Set(s.visibleDomains);
      next.has(domain) ? next.delete(domain) : next.add(domain);
      return { visibleDomains: next };
    }),

  visibleEngines: new Set<string>(),
  toggleEngine: (engineId) =>
    set((s) => {
      const next = new Set(s.visibleEngines);
      next.has(engineId) ? next.delete(engineId) : next.add(engineId);
      return { visibleEngines: next };
    }),
}));
