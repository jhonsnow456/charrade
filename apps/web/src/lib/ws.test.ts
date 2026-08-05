import { afterEach, describe, expect, it, vi } from 'vitest';
import { GameClient, wsUrl, type Socket } from './ws';

class FakeSocket implements Socket {
  readyState = 0;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;

  send(data: string): void {
    this.sent.push(data);
  }

  close(): void {
    this.readyState = 3;
  }
}

afterEach(() => {
  vi.useRealTimers();
});

describe('wsUrl', () => {
  it('builds a websocket url from the current host', () => {
    expect(wsUrl('room-1', 'p1')).toBe(
      `ws://${window.location.host}/api/rooms/room-1/ws?playerId=p1`,
    );
  });
});

describe('GameClient', () => {
  it('parses incoming messages and forwards them', () => {
    const socket = new FakeSocket();
    const onMessage = vi.fn();
    const client = new GameClient('ws://x', onMessage, {
      createSocket: () => socket,
    });

    client.connect();
    socket.onmessage?.({ data: JSON.stringify({ type: 'state', room: { id: 'r1' } }) });

    expect(onMessage).toHaveBeenCalledWith({ type: 'state', room: { id: 'r1' } });
  });

  it('routes signal messages to the onSignal handler', () => {
    const socket = new FakeSocket();
    const onMessage = vi.fn();
    const onSignal = vi.fn();
    const client = new GameClient('ws://x', onMessage, {
      createSocket: () => socket,
      onSignal,
    });

    client.connect();
    socket.onmessage?.({
      data: JSON.stringify({
        type: 'signal',
        from: 'p1',
        to: 'p2',
        payload: { type: 'offer', sdp: 'sdp-1' },
      }),
    });

    expect(onSignal).toHaveBeenCalledWith({
      type: 'signal',
      from: 'p1',
      to: 'p2',
      payload: { type: 'offer', sdp: 'sdp-1' },
    });
    expect(onMessage).not.toHaveBeenCalled();
  });

  it('serializes outgoing signal messages', () => {
    const socket = new FakeSocket();
    socket.readyState = 1;
    const client = new GameClient('ws://x', vi.fn(), { createSocket: () => socket });

    client.connect();
    client.send({ type: 'signal', to: 'p1', payload: { candidate: {} as RTCIceCandidate } });

    expect(socket.sent).toEqual([
      JSON.stringify({ type: 'signal', to: 'p1', payload: { candidate: {} } }),
    ]);
  });

  it('serializes outgoing messages when open', () => {
    const socket = new FakeSocket();
    socket.readyState = 1;
    const client = new GameClient('ws://x', vi.fn(), { createSocket: () => socket });

    client.connect();
    client.send({ type: 'guess', text: 'octopus' });

    expect(socket.sent).toEqual([JSON.stringify({ type: 'guess', text: 'octopus' })]);
  });

  it('does not send when the socket is not open', () => {
    const socket = new FakeSocket();
    const client = new GameClient('ws://x', vi.fn(), { createSocket: () => socket });

    client.connect();
    client.send({ type: 'guess', text: 'octopus' });

    expect(socket.sent).toEqual([]);
  });

  it('reconnects after a close', () => {
    vi.useFakeTimers();
    const sockets: FakeSocket[] = [];
    const client = new GameClient('ws://x', vi.fn(), {
      createSocket: () => {
        const socket = new FakeSocket();
        sockets.push(socket);
        return socket;
      },
      reconnectMs: 1000,
    });

    client.connect();
    sockets[0].onclose?.();
    expect(sockets).toHaveLength(1);

    vi.advanceTimersByTime(1000);
    expect(sockets).toHaveLength(2);
  });

  it('stops reconnecting once closed by the client', () => {
    vi.useFakeTimers();
    const sockets: FakeSocket[] = [];
    const client = new GameClient('ws://x', vi.fn(), {
      createSocket: () => {
        const socket = new FakeSocket();
        sockets.push(socket);
        return socket;
      },
      reconnectMs: 1000,
    });

    client.connect();
    client.close();
    vi.advanceTimersByTime(5000);

    expect(sockets).toHaveLength(1);
  });
});
