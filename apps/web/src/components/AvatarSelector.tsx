import { AVATAR_IDS, avatarSrc } from '../lib/avatars';
import './AvatarSelector.css';

interface AvatarSelectorProps {
  value: string;
  onChange: (id: string) => void;
}

function AvatarSelector({ value, onChange }: AvatarSelectorProps) {
  return (
    <div className="avatar-selector" role="radiogroup" aria-label="Choose an avatar">
      {AVATAR_IDS.map((id) => (
        <button
          key={id}
          type="button"
          role="radio"
          aria-checked={id === value}
          aria-label={id}
          className={id === value ? 'avatar-option selected' : 'avatar-option'}
          onClick={() => onChange(id)}
        >
          <img src={avatarSrc(id)} alt={id} />
        </button>
      ))}
    </div>
  );
}

export default AvatarSelector;
