import { useMemo } from 'react';
import { Badge, Group, ScrollArea, Stack, Table, Tabs, Text } from '@mantine/core';
import { useStore } from '../../store';
import { resultsTimeRange } from '../../lib/playback';
import type { EventType } from '../../types';

const EVENT_COLOR: Record<EventType, string> = {
  detection: 'blue',
  engagement: 'orange',
  kill: 'red',
  damage: 'yellow',
  launch: 'grape',
  waypoint_reached: 'gray',
  mission_complete: 'teal',
};

export default function ResultsPanel() {
  const results = useStore((s) => s.results);
  const activeScenario = useStore((s) => s.activeScenario);
  const entities = activeScenario?.entities ?? [];
  const playbackTimeMS = useStore((s) => s.playbackTimeMS);

  const nameById = useMemo(() => {
    const m = new Map<string, string>();
    for (const e of entities) m.set(e.id, e.name);
    return (id: string) => m.get(id) ?? id;
  }, [entities]);

  const range = useMemo(() => (results ? resultsTimeRange(results) : null), [results]);

  if (!results || !range) return null;

  const relMs = (absMs: number) => absMs - range.startMs;
  const nowMs = playbackTimeMS;

  return (
    <Tabs defaultValue="moe" h="100%" style={{ display: 'flex', flexDirection: 'column' }}>
      <Tabs.List>
        <Tabs.Tab value="moe">MOEs</Tabs.Tab>
        <Tabs.Tab value="events">
          Events
          <Badge size="xs" ml={6} variant="light">
            {results.events.length}
          </Badge>
        </Tabs.Tab>
        <Tabs.Tab value="kills">
          Kills
          <Badge size="xs" ml={6} variant="light" color="red">
            {results.kill_chains.length}
          </Badge>
        </Tabs.Tab>
      </Tabs.List>

      <Tabs.Panel value="moe" flex={1} mih={0}>
        <ScrollArea h="100%" p="sm">
          <Table withRowBorders={false} verticalSpacing={4}>
            <Table.Tbody>
              {results.moe_metrics.map((mt) => (
                <Table.Tr key={mt.key}>
                  <Table.Td>
                    <Text size="xs" c="dimmed">
                      {mt.key.replace(/_/g, ' ')}
                    </Text>
                  </Table.Td>
                  <Table.Td ta="right">
                    <Text size="sm" ff="monospace" fw={700}>
                      {mt.value}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="10px" c="dimmed">
                      {mt.unit}
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </ScrollArea>
      </Tabs.Panel>

      <Tabs.Panel value="events" flex={1} mih={0}>
        <ScrollArea h="100%" p="sm">
          <Stack gap={3}>
            {results.events.map((ev, i) => {
              const rel = relMs(ev.timestamp_ms);
              const future = rel > nowMs;
              return (
                <Group key={i} gap={6} wrap="nowrap" opacity={future ? 0.4 : 1}>
                  <Text size="10px" ff="monospace" c="dimmed" w={42} ta="right">
                    {Math.round(rel / 1000)}s
                  </Text>
                  <Badge size="xs" variant="light" color={EVENT_COLOR[ev.type] ?? 'gray'} w={120}>
                    {ev.type}
                  </Badge>
                  <Text size="xs" truncate>
                    {nameById(ev.entity_id)}
                    {ev.target_entity_id ? ` → ${nameById(ev.target_entity_id)}` : ''}
                  </Text>
                </Group>
              );
            })}
          </Stack>
        </ScrollArea>
      </Tabs.Panel>

      <Tabs.Panel value="kills" flex={1} mih={0}>
        <ScrollArea h="100%" p="sm">
          <Stack gap={4}>
            {results.kill_chains.length === 0 && (
              <Text size="xs" c="dimmed">
                No kills recorded.
              </Text>
            )}
            {results.kill_chains.map((kc, i) => (
              <Group key={i} gap={6} wrap="nowrap">
                <Text size="10px" ff="monospace" c="dimmed" w={42} ta="right">
                  {Math.round(relMs(kc.killed_at_ms) / 1000)}s
                </Text>
                <Text size="xs" c="cyan">
                  {nameById(kc.attacker_entity_id)}
                </Text>
                <Text size="xs" c="dimmed">
                  destroyed
                </Text>
                <Text size="xs" c="red">
                  {nameById(kc.target_entity_id)}
                </Text>
              </Group>
            ))}
          </Stack>
        </ScrollArea>
      </Tabs.Panel>
    </Tabs>
  );
}
