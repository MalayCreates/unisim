import { ActionIcon, Group, ScrollArea, Stack, Text, UnstyledButton } from '@mantine/core';
import { IconTrash } from '@tabler/icons-react';
import { useStore } from '../../store';
import { SIDE_CSS, ENTITY_LABEL } from '../../lib/domain';
import { entityUnitId } from '../../lib/units';
import type { Entity } from '../../types';

export default function EntityList() {
  const activeScenario = useStore((s) => s.activeScenario);
  const entities = activeScenario?.entities ?? [];
  const selectedEntityId = useStore((s) => s.selectedEntityId);
  const selectEntity = useStore((s) => s.selectEntity);
  const removeEntity = useStore((s) => s.removeEntity);
  const results = useStore((s) => s.results);

  if (entities.length === 0) {
    return (
      <Text size="xs" c="dimmed">
        No units yet. Switch to Place mode and click the map.
      </Text>
    );
  }

  const renderRow = (e: Entity) => (
    <Group
      key={e.id}
      justify="space-between"
      wrap="nowrap"
      px={6}
      py={3}
      style={{
        borderRadius: 4,
        background: selectedEntityId === e.id ? 'var(--mantine-color-dark-5)' : undefined,
      }}
    >
      <UnstyledButton onClick={() => selectEntity(e.id)} style={{ flex: 1, minWidth: 0 }}>
        <Group gap={6} wrap="nowrap">
          <span
            style={{
              width: 9,
              height: 9,
              borderRadius: '50%',
              background: SIDE_CSS[e.side],
              flexShrink: 0,
            }}
          />
          <Text size="xs" truncate>
            {e.name}
          </Text>
          <Text size="10px" c="dimmed" truncate>
            {ENTITY_LABEL[e.type]}
          </Text>
        </Group>
      </UnstyledButton>
      {!results && (
        <ActionIcon
          size="xs"
          variant="subtle"
          color="red"
          onClick={() => removeEntity(e.id)}
          aria-label="Remove unit"
        >
          <IconTrash size={13} />
        </ActionIcon>
      )}
    </Group>
  );

  // Bucket entities by unit, preserving first-seen order; ungrouped entities
  // fall into a trailing bucket. When nothing is assigned to a unit, render a
  // flat list exactly as before (no headers).
  const groupOrder: string[] = [];
  const grouped = new Map<string, Entity[]>();
  const ungrouped: Entity[] = [];
  for (const e of entities) {
    const u = entityUnitId(e);
    if (!u) {
      ungrouped.push(e);
      continue;
    }
    if (!grouped.has(u)) {
      grouped.set(u, []);
      groupOrder.push(u);
    }
    grouped.get(u)!.push(e);
  }

  return (
    <ScrollArea.Autosize mah={220}>
      {groupOrder.length === 0 ? (
        <Stack gap={2}>{entities.map(renderRow)}</Stack>
      ) : (
        <Stack gap={6}>
          {groupOrder.map((u) => (
            <div key={u}>
              <Text size="10px" fw={700} c="dimmed" tt="uppercase" px={6} mb={2}>
                {u} ({grouped.get(u)!.length})
              </Text>
              <Stack gap={2}>{grouped.get(u)!.map(renderRow)}</Stack>
            </div>
          ))}
          {ungrouped.length > 0 && (
            <div>
              <Text size="10px" fw={700} c="dimmed" tt="uppercase" px={6} mb={2}>
                Ungrouped ({ungrouped.length})
              </Text>
              <Stack gap={2}>{ungrouped.map(renderRow)}</Stack>
            </div>
          )}
        </Stack>
      )}
    </ScrollArea.Autosize>
  );
}
