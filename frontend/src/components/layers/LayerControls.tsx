import { Chip, Group, Stack, Text } from '@mantine/core';
import { useStore } from '../../store';
import { DOMAINS, DOMAIN_COLOR } from '../../lib/domain';

export default function LayerControls() {
  const visibleDomains = useStore((s) => s.visibleDomains);
  const setVisibleDomains = useStore((s) => s.setVisibleDomains);
  const results = useStore((s) => s.results);
  const visibleEngines = useStore((s) => s.visibleEngines);
  const setVisibleEngines = useStore((s) => s.setVisibleEngines);

  return (
    <Stack gap="xs">
      <Text size="xs" c="dimmed" fw={700}>
        DOMAIN LAYERS
      </Text>
      <Chip.Group
        multiple
        value={visibleDomains}
        onChange={(v) => setVisibleDomains(v as never)}
      >
        <Group gap={6}>
          {DOMAINS.map((d) => (
            <Chip key={d} value={d} size="xs" color={DOMAIN_COLOR[d]}>
              {d}
            </Chip>
          ))}
        </Group>
      </Chip.Group>

      {results && (
        <>
          <Text size="xs" c="dimmed" fw={700} mt="xs">
            ENGINE RESULTS
          </Text>
          <Chip.Group
            multiple
            value={visibleEngines}
            onChange={(v) => setVisibleEngines(v as never)}
          >
            <Group gap={6}>
              <Chip value={results.engine_id} size="xs" color="grape">
                {results.engine_id}
              </Chip>
            </Group>
          </Chip.Group>
        </>
      )}
    </Stack>
  );
}
