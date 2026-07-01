import { AppShell, Divider, Stack, Text, Center } from '@mantine/core';
import { useStore } from './store';
import TopBar from './components/TopBar';
import MapCanvas from './components/canvas/MapCanvas';
import EntityPalette from './components/canvas/EntityPalette';
import EntityList from './components/canvas/EntityList';
import LayerControls from './components/layers/LayerControls';
import MissionPanel from './components/mission/MissionPanel';
import ResultsPanel from './components/results/ResultsPanel';
import PlaybackBar from './components/playback/PlaybackBar';

const HEADER_H = 52;
const FOOTER_H = 48;

export default function App() {
  const activeScenario = useStore((s) => s.activeScenario);
  const results = useStore((s) => s.results);
  const batchResult = useStore((s) => s.batchResult);

  // Playback (scrubbing a single track) only makes sense for one run; a
  // batch aggregates many runs with no single timeline to scrub.
  const showFooter = !!results;
  const mapHeight = `calc(100vh - ${HEADER_H}px - ${showFooter ? FOOTER_H : 0}px)`;

  return (
    <AppShell
      header={{ height: HEADER_H }}
      navbar={{ width: 264, breakpoint: 'sm' }}
      aside={{ width: 344, breakpoint: 'sm' }}
      footer={showFooter ? { height: FOOTER_H } : undefined}
      padding={0}
    >
      <AppShell.Header>
        <TopBar />
      </AppShell.Header>

      <AppShell.Navbar p="sm">
        <Stack gap="sm">
          <EntityPalette />
          <Divider />
          <Text size="xs" c="dimmed" fw={700}>
            UNITS
          </Text>
          <EntityList />
          <Divider />
          <LayerControls />
        </Stack>
      </AppShell.Navbar>

      <AppShell.Aside>
        <Text size="xs" c="dimmed" fw={700} p="sm" pb={0}>
          {results || batchResult ? 'RESULTS' : 'MISSION'}
        </Text>
        {results || batchResult ? <ResultsPanel /> : <MissionPanel />}
      </AppShell.Aside>

      {showFooter && (
        <AppShell.Footer>
          <PlaybackBar />
        </AppShell.Footer>
      )}

      <AppShell.Main>
        {activeScenario ? (
          <div style={{ height: mapHeight, width: '100%' }}>
            <MapCanvas />
          </div>
        ) : (
          <Center h={mapHeight}>
            <Text c="dimmed">Select or create a scenario to begin.</Text>
          </Center>
        )}
      </AppShell.Main>
    </AppShell>
  );
}
