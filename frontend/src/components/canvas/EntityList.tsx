import { ActionIcon, Group, ScrollArea, Stack, Text, UnstyledButton } from '@mantine/core';
import { IconTrash } from '@tabler/icons-react';
import { useStore } from '../../store';
import { SIDE_CSS, ENTITY_LABEL } from '../../lib/domain';

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

  return (
    <ScrollArea.Autosize mah={220}>
      <Stack gap={2}>
        {entities.map((e) => (
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
        ))}
      </Stack>
    </ScrollArea.Autosize>
  );
}
