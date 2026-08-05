import { mapRoom, type RawRoom, type Room } from './game';

export function rawRoom(overrides: Partial<RawRoom> = {}): RawRoom {
  return {
    id: 'room-1',
    hostId: 'p1',
    phase: 'playing',
    players: [{ id: 'p1', name: 'Alice', avatar: 'avatar-1', score: 2 }],
    round: {
      actorId: 'p1',
      word: 'octopus',
      duration: 60_000_000_000,
      startedAt: '2026-01-01T00:00:00Z',
      guesses: [],
      completed: false,
    },
    ...overrides,
  };
}

export function buildRoom(overrides: Partial<Room> = {}): Room {
  return mapRoom(rawRoom(overrides as Partial<RawRoom>));
}
