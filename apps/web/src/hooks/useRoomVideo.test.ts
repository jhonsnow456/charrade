import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useRoomVideo, type RoomVideoOptions, type RoomVideoState } from './useRoomVideo';
import type { RTCFactories } from '../lib/webrtc';

class FakeTrack {
  stop = vi.fn();
}

class FakeStream {
  tracks: FakeTrack[];
  constructor(count = 1) {
    this.tracks = Array.from({ length: count }, () => new FakeTrack());
  }
  getTracks() {
    return this.tracks;
  }
}

interface FakePCInstance {
  addTrack: ReturnType<typeof vi.fn>;
  createOffer: ReturnType<typeof vi.fn>;
  createAnswer: ReturnType<typeof vi.fn>;
  setLocalDescription: ReturnType<typeof vi.fn>;
  setRemoteDescription: ReturnType<typeof vi.fn>;
  addIceCandidate: ReturnType<typeof vi.fn>;
  close: ReturnType<typeof vi.fn>;
  localDescription: { type?: string; sdp?: string } | null;
  remoteDescription: unknown;
  onicecandidate: ((event: { candidate?: unknown }) => void) | null;
  ontrack: ((event: { streams: FakeStream[]; track: FakeTrack }) => void) | null;
}

function makeFactories(instances: FakePCInstance[]) {
  const factories: RTCFactories = {
    createPeerConnection: (() => {
      const pc: FakePCInstance = {
        addTrack: vi.fn(),
        createOffer: vi.fn(async () => ({ type: 'offer', sdp: 'offer-sdp' })),
        createAnswer: vi.fn(async () => ({ type: 'answer', sdp: 'answer-sdp' })),
        setLocalDescription: vi.fn(async (d: unknown) => {
          pc.localDescription = d as { type?: string; sdp?: string };
        }),
        setRemoteDescription: vi.fn(async (d: unknown) => {
          pc.remoteDescription = d;
        }),
        addIceCandidate: vi.fn(),
        close: vi.fn(),
        localDescription: null,
        remoteDescription: null,
        onicecandidate: null,
        ontrack: null,
      };
      instances.push(pc);
      return pc;
    }) as unknown as RTCFactories['createPeerConnection'],
    getUserMedia: vi.fn(async () => new FakeStream() as unknown as MediaStream),
  };
  return factories;
}

function makeOptions(overrides: Partial<RoomVideoOptions> = {}) {
  return {
    playerId: 'p1',
    actorId: 'p1',
    peerIds: ['p2', 'p3'],
    send: vi.fn(),
    ...overrides,
  };
}

describe('useRoomVideo', () => {
  it('opens the camera and offers a connection to every viewer when acting', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const send = vi.fn();
    const { result } = renderHook(() => useRoomVideo(makeOptions({ factories, send })));

    await waitFor(() => expect(result.current.localStream).not.toBeNull());

    expect(factories.getUserMedia).toHaveBeenCalledWith({ video: true, audio: false });
    expect(instances).toHaveLength(2);
    await waitFor(() =>
      expect(send).toHaveBeenCalledWith({
        type: 'signal',
        to: 'p2',
        payload: { type: 'offer', sdp: 'offer-sdp' },
      }),
    );
    expect(send).toHaveBeenCalledWith({
      type: 'signal',
      to: 'p3',
      payload: { type: 'offer', sdp: 'offer-sdp' },
    });
  });

  it('answers the actor offer and surfaces the remote stream when viewing', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const send = vi.fn();
    const { result } = renderHook(() =>
      useRoomVideo(makeOptions({ playerId: 'p2', actorId: 'p1', factories, send })),
    );

    act(() => {
      result.current.handleSignal('p1', { type: 'offer', sdp: 'offer-1' });
    });
    await waitFor(() =>
      expect(send).toHaveBeenCalledWith({
        type: 'signal',
        to: 'p1',
        payload: { type: 'answer', sdp: 'answer-sdp' },
      }),
    );

    const remote = new FakeStream();
    act(() => {
      instances[0].ontrack?.({ streams: [remote], track: remote.tracks[0] });
    });
    await waitFor(() => expect(result.current.remoteStream).toBe(remote));
  });

  it('forwards ice candidates to the current session', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const send = vi.fn();
    const { result } = renderHook(() => useRoomVideo(makeOptions({ factories, send })));

    await waitFor(() => expect(result.current.localStream).not.toBeNull());
    const candidate = { candidate: 'ice-1' } as RTCIceCandidate;
    act(() => {
      result.current.handleSignal('p2', { candidate });
    });
    await waitFor(() => expect(instances[0].addIceCandidate).toHaveBeenCalledWith(candidate));
  });

  it('stops the camera and closes peers when the round ends', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const send = vi.fn();
    const { result, rerender } = renderHook<RoomVideoState, { actorId: string | null }>(
      ({ actorId }: { actorId: string | null }) =>
        useRoomVideo(makeOptions({ actorId, factories, send })),
      { initialProps: { actorId: 'p1' } },
    );

    await waitFor(() => expect(result.current.localStream).not.toBeNull());

    rerender({ actorId: null });
    await waitFor(() => expect(result.current.localStream).toBeNull());
    expect(instances[0].close).toHaveBeenCalled();
  });

  it('does not restart the camera across state broadcasts with the same viewers', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const send = vi.fn();
    const { rerender } = renderHook(
      ({ peerIds }: { peerIds: string[] }) =>
        useRoomVideo(makeOptions({ peerIds, factories, send })),
      { initialProps: { peerIds: ['p2', 'p3'] } },
    );

    await waitFor(() => expect(factories.getUserMedia).toHaveBeenCalledTimes(1));

    rerender({ peerIds: ['p3', 'p2'] });
    await waitFor(() => expect(factories.getUserMedia).toHaveBeenCalledTimes(1));
    expect(instances).toHaveLength(2);
  });

  it('reports a camera error instead of crashing', async () => {
    const factories: RTCFactories = {
      createPeerConnection: (() => ({
        addTrack: vi.fn(),
        createOffer: vi.fn(),
        createAnswer: vi.fn(),
        setLocalDescription: vi.fn(),
        setRemoteDescription: vi.fn(),
        addIceCandidate: vi.fn(),
        close: vi.fn(),
        localDescription: null,
        remoteDescription: null,
        onicecandidate: null,
        ontrack: null,
      })) as unknown as RTCFactories['createPeerConnection'],
      getUserMedia: vi.fn(async () => {
        throw new Error('Permission denied');
      }),
    };
    const { result } = renderHook(() => useRoomVideo(makeOptions({ factories })));

    await waitFor(() => expect(result.current.error).toBe('Permission denied'));
    expect(result.current.localStream).toBeNull();
  });
});
