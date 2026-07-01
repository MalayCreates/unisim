import {
  ActionIcon,
  Button,
  Divider,
  Group,
  NumberInput,
  ScrollArea,
  SegmentedControl,
  Select,
  Stack,
  Text,
  TextInput,
  Textarea,
} from '@mantine/core';
import { IconTrash, IconMapPin } from '@tabler/icons-react';
import { useStore } from '../../store';
import { ENTITY_TYPES, SIDES } from '../../lib/domain';
import { UNIT_ATTR } from '../../lib/units';
import type { EntityType, MissionType, ROE, Side } from '../../types';

const MISSION_TYPES: { value: MissionType; label: string }[] = [
  { value: 'cap', label: 'CAP' },
  { value: 'cas', label: 'CAS' },
  { value: 'strike', label: 'Strike' },
  { value: 'patrol', label: 'Patrol' },
  { value: 'recon', label: 'Recon' },
  { value: 'escort', label: 'Escort' },
  { value: 'sar', label: 'SAR' },
  { value: 'transit', label: 'Transit' },
];

const ROE_OPTS: { value: ROE; label: string }[] = [
  { value: 'weapons_hold', label: 'Hold' },
  { value: 'weapons_tight', label: 'Tight' },
  { value: 'weapons_free', label: 'Free' },
];

export default function MissionPanel() {
  const selectedEntityId = useStore((s) => s.selectedEntityId);
  const entity = useStore((s) =>
    s.activeScenario?.entities.find((e) => e.id === s.selectedEntityId),
  );
  const mission = useStore((s) =>
    (s.activeScenario?.missions ?? []).find((m) => m.entity_id === s.selectedEntityId),
  );
  const updateEntity = useStore((s) => s.updateEntity);
  const updateMission = useStore((s) => s.updateMission);
  const removeEntity = useStore((s) => s.removeEntity);
  const removeWaypoint = useStore((s) => s.removeWaypoint);
  const setInteractionMode = useStore((s) => s.setInteractionMode);

  if (!entity || !selectedEntityId) {
    return (
      <Text size="sm" c="dimmed" p="sm">
        Select a unit on the map or in the list to edit its mission.
      </Text>
    );
  }

  const m = mission ?? {
    entity_id: selectedEntityId,
    mission_type: 'patrol' as MissionType,
    waypoints: [],
    roe: 'weapons_tight' as ROE,
    objectives: [],
    timeline: { start_offset_s: 0, expected_duration_s: 0 },
  };

  const patchMission = (patch: Partial<typeof m>) => updateMission({ ...m, ...patch });

  const setWaypointField = (i: number, field: 'speed_ms' | 'hold_time_s', value: number) => {
    const waypoints = m.waypoints.map((w, idx) => (idx === i ? { ...w, [field]: value } : w));
    patchMission({ waypoints });
  };

  return (
    <ScrollArea.Autosize mah="calc(100vh - 160px)">
      <Stack gap="sm" p="sm">
        <Text size="xs" c="dimmed" fw={700}>
          UNIT
        </Text>
        <TextInput
          size="xs"
          label="Name"
          value={entity.name}
          onChange={(e) => updateEntity({ ...entity, name: e.currentTarget.value })}
        />
        <Select
          size="xs"
          label="Type"
          value={entity.type}
          data={ENTITY_TYPES}
          onChange={(v) => v && updateEntity({ ...entity, type: v as EntityType })}
          comboboxProps={{ withinPortal: true }}
        />
        <TextInput
          size="xs"
          label="Unit (optional)"
          placeholder="e.g. Alpha Flight"
          value={entity.attributes?.[UNIT_ATTR] ?? ''}
          onChange={(e) => {
            const v = e.currentTarget.value;
            const attributes = { ...entity.attributes };
            if (v.trim()) attributes[UNIT_ATTR] = v;
            else delete attributes[UNIT_ATTR];
            updateEntity({ ...entity, attributes });
          }}
        />
        <div>
          <Text size="xs" mb={4}>
            Side
          </Text>
          <SegmentedControl
            size="xs"
            fullWidth
            value={entity.side}
            data={SIDES.map((s) => ({ label: s.label, value: s.value }))}
            onChange={(v) => updateEntity({ ...entity, side: v as Side })}
          />
        </div>
        <Group gap={6} grow>
          <NumberInput
            size="xs"
            label="Lat"
            value={entity.position.lat}
            decimalScale={4}
            onChange={(v) =>
              updateEntity({ ...entity, position: { ...entity.position, lat: Number(v) || 0 } })
            }
          />
          <NumberInput
            size="xs"
            label="Lon"
            value={entity.position.lon}
            decimalScale={4}
            onChange={(v) =>
              updateEntity({ ...entity, position: { ...entity.position, lon: Number(v) || 0 } })
            }
          />
          <NumberInput
            size="xs"
            label="Alt (m)"
            value={entity.position.alt_m}
            onChange={(v) =>
              updateEntity({ ...entity, position: { ...entity.position, alt_m: Number(v) || 0 } })
            }
          />
        </Group>

        <Divider label="Mission" labelPosition="left" />

        <Select
          size="xs"
          label="Mission type"
          value={m.mission_type}
          data={MISSION_TYPES}
          onChange={(v) => v && patchMission({ mission_type: v as MissionType })}
          comboboxProps={{ withinPortal: true }}
        />
        <div>
          <Text size="xs" mb={4}>
            Rules of engagement
          </Text>
          <SegmentedControl
            size="xs"
            fullWidth
            value={m.roe}
            data={ROE_OPTS}
            onChange={(v) => patchMission({ roe: v as ROE })}
          />
        </div>
        <Textarea
          size="xs"
          label="Objectives (one per line)"
          autosize
          minRows={2}
          value={m.objectives.join('\n')}
          onChange={(e) =>
            patchMission({
              objectives: e.currentTarget.value.split('\n').filter((x) => x.trim() !== ''),
            })
          }
        />

        <Group justify="space-between">
          <Text size="xs" c="dimmed" fw={700}>
            WAYPOINTS ({m.waypoints.length})
          </Text>
          <Button
            size="compact-xs"
            variant="light"
            leftSection={<IconMapPin size={13} />}
            onClick={() => setInteractionMode('waypoint')}
          >
            Add via map
          </Button>
        </Group>

        {m.waypoints.length === 0 && (
          <Text size="xs" c="dimmed">
            None. Click “Add via map”, then click the map to drop waypoints.
          </Text>
        )}

        <Stack gap={6}>
          {m.waypoints.map((w, i) => (
            <Group key={i} gap={6} wrap="nowrap" align="flex-end">
              <Text size="xs" w={16} c="yellow">
                {i + 1}
              </Text>
              <NumberInput
                size="xs"
                label={i === 0 ? 'Speed m/s' : undefined}
                value={w.speed_ms}
                w={90}
                onChange={(v) => setWaypointField(i, 'speed_ms', Number(v) || 0)}
              />
              <NumberInput
                size="xs"
                label={i === 0 ? 'Hold s' : undefined}
                value={w.hold_time_s}
                w={70}
                onChange={(v) => setWaypointField(i, 'hold_time_s', Number(v) || 0)}
              />
              <Text size="10px" c="dimmed" style={{ flex: 1 }} truncate>
                {w.lat.toFixed(3)}, {w.lon.toFixed(3)}
              </Text>
              <ActionIcon
                size="sm"
                variant="subtle"
                color="red"
                onClick={() => removeWaypoint(selectedEntityId, i)}
                aria-label="Remove waypoint"
              >
                <IconTrash size={14} />
              </ActionIcon>
            </Group>
          ))}
        </Stack>

        <Divider my={4} />
        <Button
          size="xs"
          variant="light"
          color="red"
          leftSection={<IconTrash size={14} />}
          onClick={() => removeEntity(selectedEntityId)}
        >
          Remove unit
        </Button>
      </Stack>
    </ScrollArea.Autosize>
  );
}
