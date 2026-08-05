import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'motion/react';
import { getRoom } from '../lib/api';
import type { Room } from '../lib/game';
import { avatarSrc } from '../lib/avatars';
import './Score.css';

export default function Score() {
  const { roomId = '' } = useParams();
  const [room, setRoom] = useState<Room | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const data = await getRoom(roomId);
        if (!cancelled) {
          setRoom(data);
        }
      } catch {
        if (!cancelled) {
          setError('Failed to load scores');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [roomId]);

  if (loading) {
    return (
      <div className="score">
        <motion.p className="score-loading" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
          Loading scores…
        </motion.p>
      </div>
    );
  }

  if (error || !room) {
    return (
      <div className="score">
        <motion.div
          className="score-error"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
        >
          <p>{error ?? 'No scores available'}</p>
          <Link className="score-home" to="/">
            Back to home
          </Link>
        </motion.div>
      </div>
    );
  }

  const sorted = [...room.players].sort((a, b) => b.score - a.score);
  const maxScore = sorted[0]?.score ?? 0;
  const winners = sorted.filter((p) => p.score === maxScore && maxScore > 0);
  const winnerNames = winners.map((p) => p.name).join(', ');

  return (
    <div className="score">
      <motion.header
        className="score-header"
        initial={{ opacity: 0, y: -16 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <h1 className="score-title">Final Scores</h1>
        {winners.length > 0 && (
          <motion.p
            className="score-winners"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ type: 'spring', stiffness: 180, damping: 14 }}
          >
            {winners.length === 1 ? 'Winner' : 'Winners'}: {winnerNames}
          </motion.p>
        )}
      </motion.header>

      <motion.table
        className="score-table"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        <thead>
          <tr>
            <th>Player</th>
            <th className="score-col-score">Score</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((player) => (
            <tr
              key={player.id}
              className={player.score === maxScore && maxScore > 0 ? 'winner' : ''}
            >
              <td className="score-player">
                <img src={avatarSrc(player.avatar)} alt={player.name} className="score-avatar" />
                <span>{player.name}</span>
              </td>
              <td className="score-col-score">
                <motion.span
                  key={player.score}
                  initial={{ scale: 1.3 }}
                  animate={{ scale: 1 }}
                  transition={{ type: 'spring', stiffness: 300, damping: 18 }}
                >
                  {player.score}
                </motion.span>
              </td>
            </tr>
          ))}
        </tbody>
      </motion.table>

      <motion.footer
        className="score-footer"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2 }}
      >
        <Link className="score-home" to="/">
          Back to home
        </Link>
      </motion.footer>
    </div>
  );
}
