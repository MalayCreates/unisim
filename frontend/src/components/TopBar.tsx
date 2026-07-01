import { useEffect, useState } from 'react';
import {
  Badge,
  Button,
  Group,
  Indicator,
  NumberInput,
  Select,
  Text,
  TextInput,
  Tooltip,
} from '@mantine/core';
import { IconPlus, IconDeviceFloppy, IconPlayerPlayFilled, IconX } from '@tabler/icons-react';
import { useStore } from '../store';
import { scenariosApi } from '../api/client';
import { useScenarioActions } from '../hooks/useScenarioActions';
import type { RunStatus } from '../types';

const STATUS_COLOR: Record<RunStatus, string> = {
  pending: 'yellow',
  running: 'blue',
  completed: 'teal',
  failed: 'red',
};

export default function TopBar() {
  const scenarios = useStore((s) => s.scenarios);
  const setScenarios = useStore((s) => s.setScenarios);
  const activeScenario = useStore((s) => s.activeScenario);
  const setActiveScenario = useStore((s) => s.setActiveScenario);
  const dirty = useStore((s) => s.dirty);
  const runStatus = useStore((s) => s.runStatus);
  const results = useStore((s) => s.results);
  const setResults = useStore((s) => s.setResults);
  const batchResult = useStore((s) => s.batchResult);
  const setBatchResult = useStore((s) => s.setBatchResult);

  const { create, save, run, runBatch } = useScenarioActions();

  const [engines, setEngines] = useState<string[]>(['custom-engine']);
  const [engine, setEngine] = useState('custom-engine');
  const [replications, setReplications] = useState<number>(1);
  const [newName, setNewName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load scenarios + registered adapters on mount.
  useEffect(() => {
    scenariosApi.list().then(setScenarios).catch((e) => setError(e.message));
    fetch('/api/v1/adapters')
      .then((r) => r.json())
      .then((list: { engine_id: string }[]) => {
        if (Array.isArray(list) && list.length > 0) {
          const ids = list.map((a) => a.engine_id);
          setEngines(ids);
          setEngine(ids[0]);
        }
      })
      .catch(() => {});
  }, [setScenarios]);

  const guard = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await fn();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const running = runStatus === 'pending' || runStatus === 'running';

  return (
    <Group justify="space-between" h="100%" px="md" wrap="nowrap">
      <Group gap="sm" wrap="nowrap">
        <Text fw={700} ff="monospace" size="lg">
          USIP
        </Text>
        <Select
          size="xs"
          placeholder="Select scenario"
          w={220}
          searchable
          value={activeScenario?.id ?? null}
          data={scenarios.map((s) => ({ value: s.id, label: s.name }))}
          onChange={(id) => {
            const sc = scenarios.find((x) => x.id === id);
            if (sc) setActiveScenario(sc);
          }}
          comboboxProps={{ withinPortal: true }}
        />
        <TextInput
          size="xs"
          placeholder="New scenario name"
          w={160}
          value={newName}
          onChange={(e) => setNewName(e.currentTarget.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && newName.trim()) {
              guard(async () => {
                await create(newName.trim());
                setNewName('');
              });
            }
          }}
        />
        <Button
          size="xs"
          variant="default"
          leftSection={<IconPlus size={14} />}
          disabled={!newName.trim() || busy}
          onClick={() =>
            guard(async () => {
              await create(newName.trim());
              setNewName('');
            })
          }
        >
          New
        </Button>
      </Group>

      <Group gap="sm" wrap="nowrap">
        {error && (
          <Tooltip label={error} multiline w={260}>
            <Text size="xs" c="red" truncate maw={180}>
              {error}
            </Text>
          </Tooltip>
        )}

        {results || batchResult ? (
          <Button
            size="xs"
            variant="default"
            leftSection={<IconX size={14} />}
            onClick={() => {
              setResults(null);
              setBatchResult(null);
            }}
          >
            Close results
          </Button>
        ) : (
          <>
            <Indicator disabled={!dirty} color="yellow" size={8} offset={4}>
              <Button
                size="xs"
                variant="default"
                leftSection={<IconDeviceFloppy size={14} />}
                disabled={!activeScenario || !dirty || busy}
                onClick={() => guard(save)}
              >
                Save
              </Button>
            </Indicator>
            <Select
              size="xs"
              w={150}
              value={engine}
              data={engines}
              onChange={(v) => v && setEngine(v)}
              comboboxProps={{ withinPortal: true }}
            />
            <Tooltip label="Replications: run this many independent copies and aggregate MOEs (Monte Carlo)">
              <NumberInput
                size="xs"
                w={64}
                min={1}
                max={50}
                value={replications}
                onChange={(v) => setReplications(typeof v === 'number' ? v : 1)}
              />
            </Tooltip>
            <Button
              size="xs"
              color="teal"
              leftSection={<IconPlayerPlayFilled size={14} />}
              loading={running || busy}
              disabled={!activeScenario || (activeScenario.entities.length === 0)}
              onClick={() =>
                guard(() =>
                  replications > 1 ? runBatch(engine, replications) : run(engine),
                )
              }
            >
              {replications > 1 ? `Run ${replications}×` : 'Run'}
            </Button>
          </>
        )}

        {runStatus && (
          <Badge color={STATUS_COLOR[runStatus]} variant="light" size="lg">
            {batchResult && runStatus === 'running'
              ? `${batchResult.completed + batchResult.failed}/${batchResult.total}`
              : runStatus}
          </Badge>
        )}
      </Group>
    </Group>
  );
}
