export interface SignalPayload {
  type?: 'offer' | 'answer';
  sdp?: string;
  candidate?: RTCIceCandidate;
}

export interface RTCFactories {
  createPeerConnection?: (config: RTCConfiguration) => RTCPeerConnection;
  getUserMedia?: (constraints: MediaStreamConstraints) => Promise<MediaStream>;
}

const ICE_SERVERS: RTCIceServer[] = [{ urls: 'stun:stun.l.google.com:19302' }];

const turnUrl = import.meta.env.VITE_TURN_URL as string | undefined;
const turnUsername = import.meta.env.VITE_TURN_USERNAME as string | undefined;
const turnCredential = import.meta.env.VITE_TURN_CREDENTIAL as string | undefined;

if (turnUrl) {
  ICE_SERVERS.push({ urls: turnUrl, username: turnUsername, credential: turnCredential });
}

const defaultFactories: RTCFactories = {
  createPeerConnection: (config) => new RTCPeerConnection(config),
  getUserMedia: (constraints) => navigator.mediaDevices.getUserMedia(constraints),
};

/**
 * Actor side of an actor-only video broadcast: opens the camera and creates one
 * peer connection per viewer, sending SDP offers and ICE candidates over the
 * signaling channel.
 */
export class ActorBroadcast {
  private pcs = new Map<string, RTCPeerConnection>();
  private stream: MediaStream | null = null;

  constructor(
    _selfId: string,
    private readonly viewerIds: string[],
    private readonly sendSignal: (to: string, payload: SignalPayload) => void,
    private readonly factories: RTCFactories = defaultFactories,
  ) {}

  async start(): Promise<MediaStream> {
    const stream = await this.factories.getUserMedia!({ video: true, audio: false });
    this.stream = stream;
    for (const viewerId of this.viewerIds) {
      this.createPeer(viewerId, stream);
    }
    return stream;
  }

  private createPeer(viewerId: string, stream: MediaStream) {
    const pc = this.factories.createPeerConnection!({ iceServers: ICE_SERVERS });
    for (const track of stream.getTracks()) {
      pc.addTrack(track, stream);
    }
    this.pcs.set(viewerId, pc);
    pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendSignal(viewerId, { candidate: event.candidate });
      }
    };
    void pc
      .createOffer()
      .then((offer) => pc.setLocalDescription(offer))
      .then(() => {
        this.sendSignal(viewerId, {
          type: 'offer',
          sdp: pc.localDescription?.sdp,
        });
      });
  }

  handleSignal(from: string, payload: SignalPayload): void {
    const pc = this.pcs.get(from);
    if (!pc) {
      return;
    }
    if (payload.candidate) {
      void pc.addIceCandidate(payload.candidate);
      return;
    }
    if (payload.sdp) {
      void pc.setRemoteDescription({
        type: payload.type ?? 'offer',
        sdp: payload.sdp,
      });
    }
  }

  stop(): void {
    this.stream?.getTracks().forEach((track) => track.stop());
    this.pcs.forEach((pc) => pc.close());
    this.pcs.clear();
    this.stream = null;
  }
}

/**
 * Viewer side of the broadcast: reacts to the actor's offers, answers them, and
 * surfaces the incoming video stream.
 */
export class ViewerSession {
  private pc: RTCPeerConnection | null = null;

  constructor(
    _selfId: string,
    private readonly actorId: string,
    private readonly sendSignal: (to: string, payload: SignalPayload) => void,
    private readonly onStream: (stream: MediaStream) => void,
    private readonly factories: RTCFactories = defaultFactories,
  ) {}

  handleSignal(from: string, payload: SignalPayload): void {
    if (from !== this.actorId) {
      return;
    }
    if (payload.candidate) {
      void this.pc?.addIceCandidate(payload.candidate);
      return;
    }
    if (payload.sdp && payload.type === 'offer') {
      const pc = this.factories.createPeerConnection!({ iceServers: ICE_SERVERS });
      this.pc = pc;
      pc.ontrack = (event) => {
        if (event.streams.length > 0) {
          this.onStream(event.streams[0]);
        }
      };
      pc.onicecandidate = (event) => {
        if (event.candidate) {
          this.sendSignal(this.actorId, { candidate: event.candidate });
        }
      };
      void pc
        .setRemoteDescription({ type: 'offer', sdp: payload.sdp })
        .then(() => pc.createAnswer())
        .then((answer) => pc.setLocalDescription(answer))
        .then(() => {
          this.sendSignal(this.actorId, {
            type: 'answer',
            sdp: pc.localDescription?.sdp,
          });
        });
    }
  }

  stop(): void {
    this.pc?.close();
    this.pc = null;
  }
}
