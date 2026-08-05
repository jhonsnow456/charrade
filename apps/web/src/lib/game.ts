import type { SignalPayload } from './webrtc';

export type Phase = 'lobby' | 'playing' | 'finished';

export interface Player {
  id: string;
  name: string;
  avatar: string;
  score: number;
}

export interface Guess {
  playerId: string;
  text: string;
  correct: boolean;
  at: string;
}

export interface Round {
  actorId: string;
  word: string;
  durationMs: number;
  startedAt: string;
  guesses: Guess[];
  completed: boolean;
}

export interface Room {
  id: string;
  hostId: string;
  phase: Phase;
  players: Player[];
  round: Round | null;
}

export interface StateMessage {
  type: 'state';
  room: RawRoom;
}

export interface ErrorMessage {
  type: 'error';
  message: string;
}

export interface SignalMessage {
  type: 'signal';
  from: string;
  to: string;
  payload: SignalPayload;
}

export type ServerMessage = StateMessage | ErrorMessage | SignalMessage;

export type ClientMessage =
  | { type: 'start' }
  | { type: 'startRound' }
  | { type: 'endRound' }
  | { type: 'guess'; text: string }
  | { type: 'signal'; to: string; payload: SignalPayload };

export interface RawRound {
  actorId: string;
  word: string;
  duration: number;
  startedAt: string;
  guesses: Guess[];
  completed: boolean;
}

export interface RawRoom {
  id: string;
  hostId: string;
  phase: Phase;
  players: Player[];
  round: RawRound | null;
}

export function mapRoom(raw: RawRoom): Room {
  return {
    id: raw.id,
    hostId: raw.hostId,
    phase: raw.phase,
    players: raw.players,
    round: raw.round
      ? {
          actorId: raw.round.actorId,
          word: raw.round.word,
          durationMs: raw.round.duration / 1_000_000,
          startedAt: raw.round.startedAt,
          guesses: raw.round.guesses ?? [],
          completed: raw.round.completed,
        }
      : null,
  };
}
