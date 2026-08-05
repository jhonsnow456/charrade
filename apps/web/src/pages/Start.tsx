import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion } from 'motion/react';
import AvatarSelector from '../components/AvatarSelector';
import { createRoom, joinRoom } from '../lib/api';
import './Start.css';

function Start() {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [roomCode, setRoomCode] = useState('');
  const [avatar, setAvatar] = useState('avatar-1');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  function enterRoom(roomId: string, playerId: string, hostId: string) {
    sessionStorage.setItem(
      `charrade:${roomId}`,
      JSON.stringify({ playerId, hostId, name: name.trim(), avatar }),
    );
    navigate(`/room/${roomId}`);
  }

  async function handleCreate() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError('Enter your name first');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const { roomId, playerId, hostId } = await createRoom({ name: trimmed, avatar });
      enterRoom(roomId, playerId, hostId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not create the room');
      setBusy(false);
    }
  }

  async function handleJoin() {
    const trimmed = name.trim();
    const code = roomCode.trim();
    if (!trimmed) {
      setError('Enter your name first');
      return;
    }
    if (!code) {
      setError('Enter the room code to join');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const { roomId, playerId, hostId } = await joinRoom(code, { name: trimmed, avatar });
      enterRoom(roomId, playerId, hostId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not join the room');
      setBusy(false);
    }
  }

  return (
    <div className="start">
      <motion.header
        className="start-header"
        initial={{ opacity: 0, y: -16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ type: 'spring', stiffness: 120, damping: 16 }}
      >
        <Link className="start-logo" to="/">
          Charrade
        </Link>
      </motion.header>

      <motion.main
        className="start-body"
        initial="hidden"
        animate="show"
        variants={{ hidden: {}, show: { transition: { staggerChildren: 0.08 } } }}
      >
        <motion.label
          className="start-field"
          variants={{
            hidden: { opacity: 0, y: 16 },
            show: { opacity: 1, y: 0, transition: { type: 'spring', stiffness: 120, damping: 16 } },
          }}
        >
          <span className="start-label">Your name</span>
          <input
            className="start-name"
            type="text"
            placeholder="Enter Your Name"
            value={name}
            maxLength={24}
            onChange={(event) => setName(event.target.value)}
          />
        </motion.label>

        <motion.div
          className="start-field"
          variants={{
            hidden: { opacity: 0, y: 16 },
            show: { opacity: 1, y: 0, transition: { type: 'spring', stiffness: 120, damping: 16 } },
          }}
        >
          <span className="start-label">Pick an avatar</span>
          <AvatarSelector value={avatar} onChange={setAvatar} />
        </motion.div>

        <motion.div
          className="start-controls"
          variants={{
            hidden: { opacity: 0, y: 16 },
            show: { opacity: 1, y: 0, transition: { type: 'spring', stiffness: 120, damping: 16 } },
          }}
        >
          <input
            className="start-room-code"
            type="text"
            placeholder="Room code (optional)"
            value={roomCode}
            maxLength={16}
            onChange={(event) => setRoomCode(event.target.value)}
          />
          <button className="start-play" onClick={handleCreate} disabled={busy}>
            Create room
          </button>
          <button className="start-join" onClick={handleJoin} disabled={busy}>
            Join room
          </button>
          {error && <p className="start-error">{error}</p>}
        </motion.div>
      </motion.main>
    </div>
  );
}

export default Start;
