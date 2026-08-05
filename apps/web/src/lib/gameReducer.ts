import type { Room } from './game';

export interface GameState {
  room: Room | null;
  connected: boolean;
  error: string | null;
}

export type GameAction =
  | { type: 'state'; room: Room }
  | { type: 'error'; message: string }
  | { type: 'connected' }
  | { type: 'disconnected' };

export const initialGameState: GameState = {
  room: null,
  connected: false,
  error: null,
};

export function gameReducer(state: GameState, action: GameAction): GameState {
  switch (action.type) {
    case 'connected':
      return { ...state, connected: true, error: null };
    case 'disconnected':
      return { ...state, connected: false };
    case 'state':
      return { ...state, room: action.room, error: null };
    case 'error':
      return { ...state, error: action.message };
  }
}
