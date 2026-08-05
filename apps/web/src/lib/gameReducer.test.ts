import { describe, expect, it } from 'vitest';
import { gameReducer, initialGameState } from './gameReducer';
import { buildRoom } from './fixtures';

describe('gameReducer', () => {
  it('starts disconnected with no room', () => {
    expect(initialGameState).toEqual({ room: null, connected: false, error: null });
  });

  it('marks connected and clears errors', () => {
    const state = gameReducer(
      { room: null, connected: false, error: 'boom' },
      { type: 'connected' },
    );
    expect(state.connected).toBe(true);
    expect(state.error).toBeNull();
  });

  it('stores room state and clears errors', () => {
    const room = buildRoom();
    const state = gameReducer(
      { room: null, connected: true, error: 'boom' },
      {
        type: 'state',
        room,
      },
    );
    expect(state.room).toBe(room);
    expect(state.error).toBeNull();
  });

  it('keeps the last room when an error arrives', () => {
    const room = buildRoom();
    const state = gameReducer(
      { room, connected: true, error: null },
      {
        type: 'error',
        message: 'only the host can do that',
      },
    );
    expect(state.room).toBe(room);
    expect(state.error).toBe('only the host can do that');
  });

  it('tracks disconnection', () => {
    const room = buildRoom();
    const state = gameReducer(
      { room, connected: true, error: null },
      {
        type: 'disconnected',
      },
    );
    expect(state.connected).toBe(false);
    expect(state.room).toBe(room);
  });
});
