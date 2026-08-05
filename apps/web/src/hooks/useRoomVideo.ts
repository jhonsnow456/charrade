import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ClientMessage } from '../lib/game';
import {
  ActorBroadcast,
  ViewerSession,
  type RTCFactories,
  type SignalPayload,
} from '../lib/webrtc';

export interface RoomVideoOptions {
  playerId: string | null;
  actorId: string | null;
  peerIds: string[];
  send: (message: ClientMessage) => void;
  factories?: RTCFactories;
}

export interface RoomVideoState {
  error: string | null;
  localStream: MediaStream | null;
  remoteStream: MediaStream | null;
  handleSignal: (from: string, payload: SignalPayload) => void;
}

export function useRoomVideo(options: RoomVideoOptions): RoomVideoState {
  const { playerId, actorId, send, factories } = options;
  const sendRef = useRef(send);
  sendRef.current = send;
  const factoriesRef = useRef(factories);
  factoriesRef.current = factories;

  const viewerKey = useMemo(
    () =>
      options.peerIds
        .filter((id) => id !== playerId)
        .slice()
        .sort()
        .join(','),
    [options.peerIds, playerId],
  );

  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handlerRef = useRef<(from: string, payload: SignalPayload) => void>(() => {});
  const handleSignal = useCallback((from: string, payload: SignalPayload) => {
    handlerRef.current(from, payload);
  }, []);

  useEffect(() => {
    setError(null);
    setLocalStream(null);
    setRemoteStream(null);
    if (!playerId || !actorId) {
      handlerRef.current = () => {};
      return;
    }

    const viewerIds = viewerKey ? viewerKey.split(',') : [];
    const sendSignal = (to: string, payload: SignalPayload) => {
      sendRef.current({ type: 'signal', to, payload });
    };
    const deps = factoriesRef.current;

    if (actorId === playerId) {
      const actor = new ActorBroadcast(playerId, viewerIds, sendSignal, deps);
      let cancelled = false;
      handlerRef.current = (from, payload) => actor.handleSignal(from, payload);
      actor
        .start()
        .then((stream) => {
          if (!cancelled) {
            setLocalStream(stream);
          }
        })
        .catch((err) => {
          if (!cancelled) {
            setError(err instanceof Error ? err.message : 'Could not start the camera');
          }
        });
      return () => {
        cancelled = true;
        actor.stop();
      };
    }

    const viewer = new ViewerSession(
      playerId,
      actorId,
      sendSignal,
      (stream) => setRemoteStream(stream),
      deps,
    );
    handlerRef.current = (from, payload) => viewer.handleSignal(from, payload);
    return () => {
      viewer.stop();
    };
  }, [playerId, actorId, viewerKey]);

  return { error, localStream, remoteStream, handleSignal };
}
