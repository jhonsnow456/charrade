import { useEffect, useRef, useState } from 'react';
import { Link, useParams, useNavigate } from 'react-router-dom';
import { AnimatePresence, motion } from 'motion/react';
import PlayerList from '../components/PlayerList';
import { useGame, type GameSession } from '../hooks/useGame';
import { useRoomVideo } from '../hooks/useRoomVideo';
import type { ClientMessage, Player, Room } from '../lib/game';
import type { SignalPayload } from '../lib/webrtc';
import './Room.css';

const STORAGE_PREFIX = 'charrade:';

function loadSession(roomId: string): GameSession | null {
  try {
    const raw = sessionStorage.getItem(`${STORAGE_PREFIX}${roomId}`);
    return raw ? (JSON.parse(raw) as GameSession) : null;
  } catch {
    return null;
  }
}

function Room() {
  const { roomId = '' } = useParams();
  const navigate = useNavigate();
  const [session] = useState<GameSession | null>(() => loadSession(roomId));
  const signalRef = useRef<(from: string, payload: SignalPayload) => void>(() => {});
  const { state, send } = useGame(roomId, session, (message) =>
    signalRef.current(message.from, message.payload),
  );

  const round = state.room?.round ?? null;
  const roundActive = round !== null && !round.completed;
  const video = useRoomVideo({
    playerId: session?.playerId ?? null,
    actorId: roundActive ? round.actorId : null,
    peerIds: state.room?.players.map((player) => player.id) ?? [],
    send,
  });
  useEffect(() => {
    signalRef.current = video.handleSignal;
  });
  useEffect(() => {
    if (state.room?.phase === 'finished') {
      navigate(`/room/${state.room.id}/score`, { replace: true });
    }
  }, [state.room?.phase, state.room?.id, navigate]);

  if (!session) {
    return (
      <div className="room">
        <p className="room-note">No active session for this room.</p>
        <Link className="room-link" to="/start">
          Back to start
        </Link>
      </div>
    );
  }

  if (!state.room) {
    return (
      <div className="room room-connecting">
        <motion.p
          className="room-connecting-text"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
        >
          Connecting…
        </motion.p>
      </div>
    );
  }

  const room = state.room;
  const isHost = room.hostId === session.playerId;

  return (
    <div className="room">
      <header className="room-header">
        <Link className="room-leave" to="/start" aria-label="Leave room">
          ← leave
        </Link>
        <CopyCode code={room.id} />
        <span className={state.connected ? 'room-status live' : 'room-status'} role="status">
          {state.connected ? '● live' : '○ reconnecting'}
        </span>
      </header>

      {state.error && <p className="room-error">{state.error}</p>}
      {video.error && <p className="room-error">{video.error}</p>}

      <main className="room-main">
        <section className="room-stage" aria-live="polite">
          <AnimatePresence mode="wait">
            {room.phase === 'lobby' && (
              <LobbyView key="lobby" room={room} isHost={isHost} send={send} />
            )}
            {room.phase !== 'lobby' && (
              <PlayingView
                key="playing"
                room={room}
                playerId={session.playerId}
                isHost={isHost}
                send={send}
                localStream={video.localStream}
                remoteStream={video.remoteStream}
              />
            )}
          </AnimatePresence>
        </section>
        <aside className="room-side">
          <PlayerList players={room.players} actorId={room.round?.actorId} />
        </aside>
      </main>
    </div>
  );
}

function CopyCode({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      // Clipboard unavailable (non-secure context); the code is still visible.
    }
  }

  return (
    <button className="room-code" type="button" onClick={copy} aria-label="Copy room code">
      <span className="room-code-label">Room</span>
      <span className="room-code-value">{code}</span>
      {copied && (
        <motion.span
          className="room-code-copied"
          initial={{ opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
        >
          copied
        </motion.span>
      )}
    </button>
  );
}

function LobbyView({
  room,
  isHost,
  send,
}: {
  room: Room;
  isHost: boolean;
  send: (message: ClientMessage) => void;
}) {
  const ready = room.players.length >= 2;
  return (
    <motion.div
      className="lobby"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ type: 'spring', stiffness: 120, damping: 16 }}
    >
      <h2 className="stage-title">The lobby</h2>
      <p className="lobby-hint">Share your room code and wait for the team to pile in.</p>
      <p className="lobby-count" aria-live="polite">
        {room.players.length} {room.players.length === 1 ? 'player' : 'players'} inside
      </p>
      {isHost ? (
        <>
          <button
            className="stage-action"
            onClick={() => send({ type: 'start' })}
            disabled={!ready}
          >
            Start game
          </button>
          {!ready && <p className="lobby-waiting">Need at least 2 players to start.</p>}
        </>
      ) : (
        <p className="lobby-waiting">
          Waiting for the host to start
          <motion.span
            aria-hidden="true"
            animate={{ opacity: [0.2, 1, 0.2] }}
            transition={{ duration: 1.4, repeat: Infinity, ease: 'easeInOut' }}
          >
            …
          </motion.span>
        </p>
      )}
    </motion.div>
  );
}

function PlayingView({
  room,
  playerId,
  isHost,
  send,
  localStream,
  remoteStream,
}: {
  room: Room;
  playerId: string;
  isHost: boolean;
  send: (message: ClientMessage) => void;
  localStream: MediaStream | null;
  remoteStream: MediaStream | null;
}) {
  const round = room.round;
  const active = round !== null && !round.completed;

  if (!round || !active) {
    const correct = round?.guesses.some((g) => g.correct) ?? false;
    return (
      <motion.div
        className="round-end"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -20 }}
        transition={{ type: 'spring', stiffness: 120, damping: 16 }}
      >
        <h2 className="stage-title">Round over</h2>
        <p className="round-result">
          {correct ? 'Someone got it!' : 'No correct guess this round.'}
        </p>
        {room.phase === 'finished' && (
          <p className="round-final">The game is finished. Redirecting to scores…</p>
        )}
      </motion.div>
    );
  }

  const actor = room.players.find((p) => p.id === round.actorId);
  return (
    <motion.div
      className="round-active"
      initial={{ opacity: 0, scale: 0.96 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.96 }}
      transition={{ type: 'spring', stiffness: 120, damping: 16 }}
    >
      {round.actorId === playerId ? (
        <ActorView round={round} actor={actor} localStream={localStream} />
      ) : (
        <GuesserView
          round={round}
          actor={actor}
          playerId={playerId}
          players={room.players}
          send={send}
          remoteStream={remoteStream}
        />
      )}
      {isHost && (
        <button className="round-end-btn" onClick={() => send({ type: 'endRound' })}>
          End round
        </button>
      )}
    </motion.div>
  );
}

function useCountdown(startedAt: string, durationMs: number): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(id);
  }, []);
  const elapsed = now - new Date(startedAt).getTime();
  return Math.max(0, Math.ceil((durationMs - elapsed) / 1000));
}

function Countdown({ startedAt, durationMs }: { startedAt: string; durationMs: number }) {
  const remaining = useCountdown(startedAt, durationMs);
  const fraction = remaining / Math.max(1, durationMs / 1000);
  const urgent = remaining <= 10;
  const circumference = 2 * Math.PI * 42;

  return (
    <div className="timer" role="timer" aria-label={`${remaining} seconds left`}>
      <svg className="timer-ring" viewBox="0 0 100 100" aria-hidden="true">
        <circle className="timer-track" cx="50" cy="50" r="42" />
        <motion.circle
          className={urgent ? 'timer-fill urgent' : 'timer-fill'}
          cx="50"
          cy="50"
          r="42"
          strokeDasharray={circumference}
          initial={false}
          animate={{ strokeDashoffset: circumference * (1 - fraction) }}
          transition={{ ease: 'linear', duration: 0.25 }}
        />
      </svg>
      <span className={urgent ? 'timer-number urgent' : 'timer-number'}>{remaining}</span>
    </div>
  );
}

function ActorView({
  round,
  actor,
  localStream,
}: {
  round: NonNullable<Room['round']>;
  actor?: Player;
  localStream: MediaStream | null;
}) {
  return (
    <div className="actor-view">
      <p className="round-role">{actor?.name}, act it out silently</p>
      <Countdown startedAt={round.startedAt} durationMs={round.durationMs} />
      <VideoFeed stream={localStream} muted placeholder="Camera starting…" />
      <motion.p
        key={round.word}
        className="round-word"
        initial={{ opacity: 0, scale: 1.4 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ type: 'spring', stiffness: 180, damping: 14 }}
      >
        {round.word}
      </motion.p>
      <p className="round-hint">Only you can see this word. Teammates must guess it.</p>
    </div>
  );
}

function GuesserView({
  round,
  actor,
  playerId,
  players,
  send,
  remoteStream,
}: {
  round: NonNullable<Room['round']>;
  actor?: Player;
  playerId: string;
  players: Player[];
  send: (message: ClientMessage) => void;
  remoteStream: MediaStream | null;
}) {
  const [guess, setGuess] = useState('');
  const [justGuessed, setJustGuessed] = useState(false);

  function playerName(id: string): string {
    return players.find((p) => p.id === id)?.name ?? 'Someone';
  }

  function submit(event: React.FormEvent) {
    event.preventDefault();
    const text = guess.trim();
    if (!text) {
      return;
    }
    send({ type: 'guess', text });
    setGuess('');
    setJustGuessed(true);
    window.setTimeout(() => setJustGuessed(false), 400);
  }

  return (
    <div className="guesser-view">
      <p className="round-role">{actor?.name} is acting</p>
      <Countdown startedAt={round.startedAt} durationMs={round.durationMs} />
      <VideoFeed stream={remoteStream} placeholder="Waiting for the actor’s camera…" />
      <form className="guess-form" onSubmit={submit}>
        <input
          className="guess-input"
          value={guess}
          maxLength={64}
          placeholder="Your guess…"
          aria-label="Your guess"
          onChange={(event) => setGuess(event.target.value)}
        />
        <motion.button
          className="guess-submit"
          type="submit"
          whileTap={{ scale: 0.94 }}
          animate={justGuessed ? { scale: [1, 1.06, 1] } : {}}
          transition={{ duration: 0.3 }}
        >
          Guess
        </motion.button>
      </form>
      {round.guesses.length > 0 && (
        <ul className="guess-log" aria-label="Guesses">
          <AnimatePresence initial={false}>
            {round.guesses.map((g, i) => (
              <motion.li
                key={`${g.playerId}-${i}`}
                className={g.correct ? 'guess correct' : 'guess'}
                initial={{ opacity: 0, x: -16 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0 }}
                layout
                transition={{ type: 'spring', stiffness: 200, damping: 22 }}
              >
                <span className="guess-player">{playerName(g.playerId)}</span>
                <span className="guess-text">{g.text}</span>
                {g.playerId === playerId && <span className="guess-mine">(you)</span>}
                {g.correct && <span className="guess-correct">✓</span>}
              </motion.li>
            ))}
          </AnimatePresence>
        </ul>
      )}
    </div>
  );
}

function VideoFeed({
  stream,
  muted,
  placeholder,
}: {
  stream: MediaStream | null;
  muted?: boolean;
  placeholder: string;
}) {
  const ref = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const element = ref.current;
    if (element && stream) {
      element.srcObject = stream;
    }
    return () => {
      if (element) {
        element.srcObject = null;
      }
    };
  }, [stream]);

  return (
    <div className={stream ? 'video-feed' : 'video-feed empty'}>
      <video ref={ref} autoPlay playsInline muted={muted} aria-label="Actor camera feed" />
      {!stream && <p className="video-placeholder">{placeholder}</p>}
    </div>
  );
}

export default Room;
