import type { Entity, EntityMission, Run, Scenario, SimResults } from '../types';

// Base URL is configurable so Electron can point it at localhost.
const BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '/api/v1';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error ?? res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

// ---- Scenarios ----

export const scenariosApi = {
  list: (): Promise<Scenario[]> => request('/scenarios'),

  get: (id: string): Promise<Scenario> => request(`/scenarios/${id}`),

  create: (data: {
    name: string;
    description?: string;
    entities?: Entity[];
    missions?: EntityMission[];
    engine_hint?: string;
    duration_s?: number;
  }): Promise<Scenario> =>
    request('/scenarios', { method: 'POST', body: JSON.stringify(data) }),

  update: (id: string, patch: Partial<Scenario>): Promise<Scenario> =>
    request(`/scenarios/${id}`, { method: 'PUT', body: JSON.stringify(patch) }),

  delete: (id: string): Promise<void> =>
    request(`/scenarios/${id}`, { method: 'DELETE' }),
};

// ---- Runs ----

export const runsApi = {
  list: (scenarioId: string): Promise<Run[]> =>
    request(`/scenarios/${scenarioId}/runs`),

  trigger: (scenarioId: string, engineId?: string): Promise<Run> =>
    request(`/scenarios/${scenarioId}/runs`, {
      method: 'POST',
      body: JSON.stringify({ engine_id: engineId ?? 'custom-engine' }),
    }),

  get: (runId: string): Promise<Run> => request(`/runs/${runId}`),

  getResults: (runId: string): Promise<SimResults> =>
    request(`/runs/${runId}/results`),
};
