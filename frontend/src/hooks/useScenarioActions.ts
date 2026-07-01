import { useCallback } from 'react';
import { batchesApi, scenariosApi, runsApi } from '../api/client';
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

  // runBatch saves any pending edits, then triggers `count` independent
  // replications of the scenario and polls until every replication finishes,
  // loading the cross-run aggregated MOEs into the store when done.
  const runBatch = useCallback(
    async (engineId: string, count: number): Promise<void> => {
      const st = useStore.getState();
      let sc = st.activeScenario;
      if (!sc) return;
      if (st.dirty) {
        await save();
        sc = useStore.getState().activeScenario!;
      }

      st.setRunStatus('pending');
      const { batch_id } = await batchesApi.create(sc.id, engineId, count);

      return new Promise((resolve, reject) => {
        const poll = async () => {
          try {
            const summary = await batchesApi.get(batch_id);
            useStore.getState().setBatchResult(summary);
            const finished = summary.completed + summary.failed;
            if (finished >= summary.total) {
              useStore.getState().setRunStatus(summary.failed > 0 ? 'failed' : 'completed');
              resolve();
            } else {
              useStore.getState().setRunStatus('running');
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

  return { create, save, run, runBatch };
}
