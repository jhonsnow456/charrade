import { afterEach, describe, expect, it, vi } from 'vitest';
import { createRoom, getRoom, joinRoom } from './api';
import { rawRoom } from './fixtures';

function stubFetch(json: unknown, ok = true) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => ({
      ok,
      status: ok ? 200 : 400,
      json: async () => json,
    })),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('api', () => {
  it('createRoom posts player info and returns ids', async () => {
    stubFetch({ roomId: 'r1', playerId: 'p1', hostId: 'p1' });

    const result = await createRoom({ name: 'Alice', avatar: 'avatar-1' });

    expect(result).toEqual({ roomId: 'r1', playerId: 'p1', hostId: 'p1' });
    expect(fetch).toHaveBeenCalledWith(
      '/api/rooms',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'Alice', avatar: 'avatar-1' }),
      }),
    );
  });

  it('joinRoom posts to the room players endpoint', async () => {
    stubFetch({ roomId: 'r1', playerId: 'p2', hostId: 'p1' });

    await joinRoom('r1', { name: 'Bob', avatar: 'avatar-2' });

    expect(fetch).toHaveBeenCalledWith('/api/rooms/r1/players', expect.anything());
  });

  it('getRoom maps the raw room state', async () => {
    stubFetch({ type: 'state', room: rawRoom() });

    const room = await getRoom('r1');

    expect(room.round?.durationMs).toBe(60_000);
    expect(room.id).toBe('room-1');
  });

  it('throws when the server returns an error', async () => {
    stubFetch({ message: 'bad' }, false);

    await expect(createRoom({ name: 'Alice', avatar: 'avatar-1' })).rejects.toThrow();
    await expect(getRoom('nope')).rejects.toThrow();
  });
});
