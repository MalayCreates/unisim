import { useEffect, useMemo } from 'react';
import { ActionIcon, Group, Slider, Text } from '@mantine/core';
import { IconPlayerPlay, IconPlayerPause, IconPlayerSkipBack } from '@tabler/icons-react';
import { useStore } from '../../store';
import { resultsTimeRange } from '../../lib/playback';

const TICK_MS = 100; // playback refresh interval
const SPEED = 30; // sim-seconds advanced per real second during playback

export default function PlaybackBar() {
  const results = useStore((s) => s.results);
  const playing = useStore((s) => s.playing);
  const setPlaying = useStore((s) => s.setPlaying);
  const playbackTimeMS = useStore((s) => s.playbackTimeMS);
  const setPlaybackTimeMS = useStore((s) => s.setPlaybackTimeMS);

  const range = useMemo(() => (results ? resultsTimeRange(results) : null), [results]);

  // Advance the clock while playing.
  useEffect(() => {
    if (!playing || !range) return;
    const id = setInterval(() => {
      const cur = useStore.getState().playbackTimeMS;
      const next = cur + TICK_MS * SPEED;
      if (next >= range.durationMs) {
        setPlaybackTimeMS(range.durationMs);
        setPlaying(false);
      } else {
        setPlaybackTimeMS(next);
      }
    }, TICK_MS);
    return () => clearInterval(id);
  }, [playing, range, setPlaybackTimeMS, setPlaying]);

  if (!results || !range) return null;

  const secs = Math.round(playbackTimeMS / 1000);
  const totalSecs = Math.round(range.durationMs / 1000);

  return (
    <Group gap="sm" px="md" py={6} wrap="nowrap" h={48}>
      <ActionIcon
        variant="subtle"
        onClick={() => {
          setPlaybackTimeMS(0);
          setPlaying(false);
        }}
        aria-label="Restart"
      >
        <IconPlayerSkipBack size={18} />
      </ActionIcon>
      <ActionIcon
        variant="filled"
        onClick={() => {
          if (playbackTimeMS >= range.durationMs) setPlaybackTimeMS(0);
          setPlaying(!playing);
        }}
        aria-label={playing ? 'Pause' : 'Play'}
      >
        {playing ? <IconPlayerPause size={18} /> : <IconPlayerPlay size={18} />}
      </ActionIcon>
      <Slider
        flex={1}
        min={0}
        max={range.durationMs || 1}
        value={playbackTimeMS}
        onChange={(v) => {
          setPlaying(false);
          setPlaybackTimeMS(v);
        }}
        label={(v) => `${Math.round(v / 1000)}s`}
      />
      <Text size="xs" ff="monospace" c="dimmed" w={70} ta="right">
        {secs}s / {totalSecs}s
      </Text>
    </Group>
  );
}
