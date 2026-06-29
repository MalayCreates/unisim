import { useEffect, useState } from 'react';
import { scenariosApi } from './api/client';
import { useStore } from './store';
import MapCanvas from './components/canvas/MapCanvas';
import type { Scenario } from './types';

export default function App() {
  const { scenarios, setScenarios, activeScenario, setActiveScenario } = useStore();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    scenariosApi.list()
      .then(setScenarios)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [setScenarios]);

  const createScenario = async () => {
    try {
      const sc = await scenariosApi.create({
        name: `Scenario ${Date.now()}`,
        description: '',
        entities: [],
        missions: [],
        engine_hint: 'custom-engine',
        duration_s: 3600,
      });
      setScenarios([...scenarios, sc]);
      setActiveScenario(sc);
    } catch (e: any) {
      setError(e.message);
    }
  };

  const selectScenario = (sc: Scenario) => setActiveScenario(sc);

  return (
    <div style={{ display: 'flex', height: '100vh', background: '#0d0d0d', color: '#eee', fontFamily: 'monospace' }}>
      {/* Sidebar */}
      <aside style={{ width: 240, padding: 12, borderRight: '1px solid #333', display: 'flex', flexDirection: 'column', gap: 8, overflow: 'auto' }}>
        <h1 style={{ margin: 0, fontSize: 14, letterSpacing: 1 }}>USIP</h1>
        <hr style={{ borderColor: '#333' }} />

        <button
          onClick={createScenario}
          style={{ background: '#1a4a7a', border: 'none', color: '#fff', padding: '6px 10px', cursor: 'pointer', borderRadius: 3 }}
        >
          + New Scenario
        </button>

        {loading && <div style={{ fontSize: 11, color: '#888' }}>Loading...</div>}
        {error && <div style={{ fontSize: 11, color: '#e55' }}>{error}</div>}

        <div style={{ fontSize: 11, color: '#888', marginTop: 4 }}>SCENARIOS</div>
        {scenarios.map((sc) => (
          <button
            key={sc.id}
            onClick={() => selectScenario(sc)}
            style={{
              background: activeScenario?.id === sc.id ? '#1a3a5a' : 'transparent',
              border: '1px solid #333',
              color: '#ddd',
              padding: '6px 8px',
              cursor: 'pointer',
              borderRadius: 3,
              textAlign: 'left',
              fontSize: 12,
            }}
          >
            {sc.name}
          </button>
        ))}

        {activeScenario && (
          <>
            <hr style={{ borderColor: '#333' }} />
            <div style={{ fontSize: 11, color: '#888' }}>ACTIVE</div>
            <div style={{ fontSize: 12 }}>{activeScenario.name}</div>
            <div style={{ fontSize: 11, color: '#888' }}>
              {activeScenario.entities?.length ?? 0} entities
            </div>
            <div style={{ fontSize: 11, color: '#888' }}>
              Engine: {activeScenario.engine_hint || 'custom-engine'}
            </div>
          </>
        )}
      </aside>

      {/* Map */}
      <main style={{ flex: 1, position: 'relative' }}>
        {activeScenario ? (
          <MapCanvas />
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#555', fontSize: 14 }}>
            Select or create a scenario to begin
          </div>
        )}
      </main>
    </div>
  );
}
