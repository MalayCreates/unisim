import { useCallback } from 'react';
import { scenariosApi, runsApi } from '../api/client';
import { useStore } from '../store';
import type { Scenario } from '../types';

// useScenarioActions centralizes create/save/run so the UI components stay thin.
export function useScenarioActions() {
  const create = useCallback(async (name: string): Promise<Scenario> => {
    const sc = await scenariosApi.create({
      name,
      description: '',
      entities: [],
      missions: [],
      engine_hint: 'custom-engine',
      duration_s: 1200,
    });
    const st = useStore.getState();
    st.upsertScenario(sc);
    st.setActiveScenario(sc);
    return sc;
  }, []);

  const save = useCallback(async (): Promise<Scenario | null> => {
    const st = useStore.getState();
    const sc = st.activeScenario;
    if (!sc) return null;
    const updated = await scenariosApi.update(sc.id, sc);
    st.upsertScenario(updated);
    // Preserve current local edits as the canonical, now-clean state.
    st.markClean();
    return updated;
  }, []);

  // run saves any pending edits, triggers a run, and polls to completion,
  // loading results into the store when done.
  const run = useCallback(
    async (engineId: string): Promise<void> => {
      const st = useStore.getState();
      let sc = st.activeScenario;
      if (!sc) return;
      if (st.dirty) {
        await save();
        sc = useStore.getState().activeScenario!;
      }

      st.setRunStatus('pending');
      const r = await runsApi.trigger(sc.id, engineId);
      st.setActiveRunId(r.id);

      return new Promise((resolve, reject) => {
        const poll = async () => {
          try {
            const cur = await runsApi.get(r.id);
            useStore.getState().setRunStatus(cur.status);
            if (cur.status === 'completed') {
              const res = await runsApi.getResults(r.id);
              useStore.getState().setResults(res);
              resolve();
            } else if (cur.status === 'failed') {
              reject(new Error(cur.error || 'run failed'));
            } else {
              setTimeout(poll, 500);
            }
          } catch (err) {
            reject(err as Error);
          }
        };
        poll();
      });
    },
    [save],
  );

  return { create, save, run };
}
