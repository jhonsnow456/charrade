import { motion } from 'motion/react';
import { avatarSrc } from '../lib/avatars';
import type { Player } from '../lib/game';
import './PlayerList.css';

interface PlayerListProps {
  players: Player[];
  actorId?: string;
}

function PlayerList({ players, actorId }: PlayerListProps) {
  return (
    <ul className="player-list" aria-label="Players">
      {players.map((player) => (
        <motion.li
          key={player.id}
          className={player.id === actorId ? 'player is-actor' : 'player'}
          layout
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          transition={{ type: 'spring', stiffness: 200, damping: 22 }}
        >
          <img className="player-avatar" src={avatarSrc(player.avatar)} alt={player.name} />
          <span className="player-name">{player.name}</span>
          {player.id === actorId && <span className="player-badge">acting</span>}
          <motion.span
            className="player-score"
            key={player.score}
            initial={{ scale: 1.35 }}
            animate={{ scale: 1 }}
            transition={{ type: 'spring', stiffness: 300, damping: 18 }}
          >
            {player.score}
          </motion.span>
        </motion.li>
      ))}
    </ul>
  );
}

export default PlayerList;
