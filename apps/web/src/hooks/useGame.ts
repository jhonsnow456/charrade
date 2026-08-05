import { useEffect, useMemo, useReducer, useRef } from 'react';
import { mapRoom, type ClientMessage, type SignalMessage } from '../lib/game';
import { gameReducer, initialGameState, type GameState } from '../lib/gameReducer';
import { GameClient, wsUrl } from '../lib/ws';

export interface GameSession {
  playerId: string;
  hostId: string;
  name: string;
  avatar: string;
}

export interface UseGame {
  state: GameState;
  send: (message: ClientMessage) => void;
}

export function useGame(
  roomId: string,
  session: GameSession | null,
  onSignal?: (message: SignalMessage) => void,
): UseGame {
  const [state, dispatch] = useReducer(gameReducer, initialGameState);
  const sessionRef = useRef(session);
  sessionRef.current = session;
  const onSignalRef = useRef(onSignal);
  onSignalRef.current = onSignal;

  const playerId = session?.playerId ?? null;
  const client = useMemo(() => {
    if (!playerId) {
      return null;
    }
    return new GameClient(
      wsUrl(roomId, playerId),
      (message) => {
        if (message.type === 'state') {
          dispatch({ type: 'state', room: mapRoom(message.room) });
        } else if (message.type === 'error') {
          dispatch({ type: 'error', message: message.message });
        }
      },
      {
        onOpen: () => dispatch({ type: 'connected' }),
        onClose: () => dispatch({ type: 'disconnected' }),
        onSignal: (message) => onSignalRef.current?.(message),
      },
    );
  }, [roomId, playerId]);

  useEffect(() => {
    if (!client) {
      return;
    }
    client.connect();
    return () => client.close();
  }, [client]);

  return { state, send: client ? client.send.bind(client) : () => {} };
}
