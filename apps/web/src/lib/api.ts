import { mapRoom, type Room } from './game';

const API_BASE = '/api';

export interface CreateRoomInput {
  name: string;
  avatar: string;
}

export interface CreateRoomResult {
  roomId: string;
  playerId: string;
  hostId: string;
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(`request failed: ${res.status}`);
  }
  return (await res.json()) as T;
}

export function createRoom(input: CreateRoomInput): Promise<CreateRoomResult> {
  return post('/rooms', input);
}

export function joinRoom(roomId: string, input: CreateRoomInput): Promise<CreateRoomResult> {
  return post(`/rooms/${roomId}/players`, input);
}

export async function getRoom(roomId: string): Promise<Room> {
  const res = await fetch(`${API_BASE}/rooms/${roomId}`);
  if (!res.ok) {
    throw new Error(`request failed: ${res.status}`);
  }
  const raw = (await res.json()) as { room: unknown };
  return mapRoom(raw.room as Parameters<typeof mapRoom>[0]);
}
