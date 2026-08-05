import { describe, expect, it } from 'vitest';
import { mapRoom, type RawRoom } from './game';
import { rawRoom } from './fixtures';

describe('mapRoom', () => {
  it('converts round duration from nanoseconds to milliseconds', () => {
    const room = mapRoom(rawRoom());
    expect(room.round?.durationMs).toBe(60_000);
  });

  it('preserves the round word', () => {
    const room = mapRoom(rawRoom());
    expect(room.round?.word).toBe('octopus');
  });

  it('passes through a null round', () => {
    const room = mapRoom(rawRoom({ round: null }));
    expect(room.round).toBeNull();
  });

  it('treats a null guesses array as empty', () => {
    const raw = rawRoom();
    raw.round = { ...raw.round, guesses: null } as unknown as RawRoom['round'];
    const room = mapRoom(raw);
    expect(room.round?.guesses).toEqual([]);
  });

  it('keeps players and phase intact', () => {
    const room = mapRoom(rawRoom());
    expect(room.phase).toBe('playing');
    expect(room.players).toHaveLength(1);
    expect(room.players[0].score).toBe(2);
  });
});

export type { RawRoom };
