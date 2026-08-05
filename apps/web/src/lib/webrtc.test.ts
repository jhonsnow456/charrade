import { describe, expect, it, vi } from 'vitest';
import { ActorBroadcast, ViewerSession, type SignalPayload, type RTCFactories } from './webrtc';

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

function waitMicrotasks() {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

describe('ActorBroadcast', () => {
  it('opens the camera and creates one peer connection per viewer', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const sendSignal = vi.fn();

    const actor = new ActorBroadcast('actor-1', ['viewer-1', 'viewer-2'], sendSignal, factories);
    const stream = await actor.start();

    expect(factories.getUserMedia).toHaveBeenCalledWith({ video: true, audio: false });
    expect((stream as unknown as FakeStream).tracks).toHaveLength(1);
    expect(instances).toHaveLength(2);
    expect(instances[0].addTrack).toHaveBeenCalled();
    expect(instances[1].addTrack).toHaveBeenCalled();
  });

  it('sends an offer to each viewer', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const sendSignal = vi.fn();

    const actor = new ActorBroadcast('actor-1', ['viewer-1', 'viewer-2'], sendSignal, factories);
    await actor.start();
    await waitMicrotasks();

    expect(sendSignal).toHaveBeenCalledWith('viewer-1', {
      type: 'offer',
      sdp: 'offer-sdp',
    });
    expect(sendSignal).toHaveBeenCalledWith('viewer-2', {
      type: 'offer',
      sdp: 'offer-sdp',
    });
  });

  it('applies answers and ice candidates to the right peer', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const actor = new ActorBroadcast('actor-1', ['viewer-1'], vi.fn(), factories);
    await actor.start();

    const answer: SignalPayload = { type: 'answer', sdp: 'answer-1' };
    actor.handleSignal('viewer-1', answer);
    expect(instances[0].setRemoteDescription).toHaveBeenCalledWith({
      type: 'answer',
      sdp: 'answer-1',
    });

    const candidate = { candidate: 'candidate-1' } as RTCIceCandidate;
    actor.handleSignal('viewer-1', { candidate });
    expect(instances[0].addIceCandidate).toHaveBeenCalledWith(candidate);
  });

  it('ignores signals from unknown viewers', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const actor = new ActorBroadcast('actor-1', ['viewer-1'], vi.fn(), factories);
    await actor.start();

    actor.handleSignal('stranger', { type: 'answer', sdp: 'x' });
    expect(instances[0].setRemoteDescription).not.toHaveBeenCalled();
  });

  it('relays its own ice candidates through the signal channel', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const sendSignal = vi.fn();
    const actor = new ActorBroadcast('actor-1', ['viewer-1'], sendSignal, factories);
    await actor.start();

    instances[0].onicecandidate?.({ candidate: { candidate: 'ice-1' } });
    expect(sendSignal).toHaveBeenCalledWith('viewer-1', { candidate: { candidate: 'ice-1' } });

    instances[0].onicecandidate?.({ candidate: null });
    expect(sendSignal).toHaveBeenCalledTimes(1);
  });

  it('stops tracks and closes peers on stop', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const actor = new ActorBroadcast('actor-1', ['viewer-1'], vi.fn(), factories);
    const stream = (await actor.start()) as unknown as FakeStream;

    actor.stop();

    expect(stream.tracks[0].stop).toHaveBeenCalled();
    expect(instances[0].close).toHaveBeenCalled();
  });
});

describe('ViewerSession', () => {
  it('answers an offer and forwards the received stream', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const sendSignal = vi.fn();
    const onStream = vi.fn();

    const viewer = new ViewerSession('viewer-1', 'actor-1', sendSignal, onStream, factories);
    viewer.handleSignal('actor-1', { type: 'offer', sdp: 'offer-1' });
    await waitMicrotasks();

    expect(instances[0].setRemoteDescription).toHaveBeenCalledWith({
      type: 'offer',
      sdp: 'offer-1',
    });
    expect(sendSignal).toHaveBeenCalledWith('actor-1', { type: 'answer', sdp: 'answer-sdp' });

    const remote = new FakeStream();
    instances[0].ontrack?.({ streams: [remote], track: remote.tracks[0] });
    expect(onStream).toHaveBeenCalledWith(remote);
  });

  it('applies ice candidates from the actor', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const viewer = new ViewerSession('viewer-1', 'actor-1', vi.fn(), vi.fn(), factories);
    viewer.handleSignal('actor-1', { type: 'offer', sdp: 'offer-1' });
    await waitMicrotasks();

    const candidate = { candidate: 'ice-1' } as RTCIceCandidate;
    viewer.handleSignal('actor-1', { candidate });
    expect(instances[0].addIceCandidate).toHaveBeenCalledWith(candidate);
  });

  it('ignores offers from anyone but the actor', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const viewer = new ViewerSession('viewer-1', 'actor-1', vi.fn(), vi.fn(), factories);

    viewer.handleSignal('imposter', { type: 'offer', sdp: 'offer-1' });

    expect(instances).toHaveLength(0);
  });

  it('closes its peer connection on stop', async () => {
    const instances: FakePCInstance[] = [];
    const factories = makeFactories(instances);
    const viewer = new ViewerSession('viewer-1', 'actor-1', vi.fn(), vi.fn(), factories);
    viewer.handleSignal('actor-1', { type: 'offer', sdp: 'offer-1' });
    await waitMicrotasks();

    viewer.stop();
    expect(instances[0].close).toHaveBeenCalled();
  });
});
