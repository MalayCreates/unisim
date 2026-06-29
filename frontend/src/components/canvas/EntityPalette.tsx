import { SegmentedControl, Select, Stack, Text, Group } from '@mantine/core';
import { useStore } from '../../store';
import { ENTITY_TYPES, SIDES } from '../../lib/domain';
import type { EntityType, Side } from '../../types';

export default function EntityPalette() {
  const interactionMode = useStore((s) => s.interactionMode);
  const setInteractionMode = useStore((s) => s.setInteractionMode);
  const paletteType = useStore((s) => s.paletteType);
  const paletteSide = useStore((s) => s.paletteSide);
  const setPalette = useStore((s) => s.setPalette);
  const selectedEntityId = useStore((s) => s.selectedEntityId);
  const results = useStore((s) => s.results);

  const disabled = !!results; // editing disabled while viewing results

  return (
    <Stack gap="xs">
      <Text size="xs" c="dimmed" fw={700}>
        MAP MODE
      </Text>
      <SegmentedControl
        size="xs"
        fullWidth
        value={interactionMode}
        onChange={(v) => setInteractionMode(v as never)}
        disabled={disabled}
        data={[
          { label: 'Select', value: 'select' },
          { label: 'Place', value: 'place' },
          { label: 'Waypoint', value: 'waypoint', disabled: !selectedEntityId },
        ]}
      />
      {interactionMode === 'place' && (
        <Text size="xs" c="cyan">
          Click the map to place a {paletteType.replace('_', ' ')}.
        </Text>
      )}
      {interactionMode === 'waypoint' && (
        <Text size="xs" c="yellow">
          Click the map to add waypoints to the selected entity.
        </Text>
      )}

      <Text size="xs" c="dimmed" fw={700} mt="xs">
        UNIT TO PLACE
      </Text>
      <Select
        size="xs"
        value={paletteType}
        onChange={(v) => v && setPalette(v as EntityType, paletteSide)}
        data={ENTITY_TYPES}
        disabled={disabled}
        comboboxProps={{ withinPortal: true }}
      />
      <Group gap={4} grow>
        <SegmentedControl
          size="xs"
          value={paletteSide}
          onChange={(v) => setPalette(paletteType, v as Side)}
          disabled={disabled}
          data={SIDES.map((s) => ({ label: s.label, value: s.value }))}
        />
      </Group>
    </Stack>
  );
}
