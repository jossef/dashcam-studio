import { t } from '../../i18n/he';
import { useApp } from '../../store/app';
import { Icon } from '../Icon';

export function StarButton({ id, className = '' }: { id: string; className?: string }) {
  const on = useApp((s) => s.favorites.has(id));
  return (
    <span
      role="button"
      tabIndex={0}
      aria-pressed={on}
      title={on ? t.unstar : t.star}
      aria-label={on ? t.unstar : t.star}
      className={`star ${on ? 'on' : ''} ${className}`}
      onClick={(e) => {
        e.stopPropagation();
        useApp.getState().setFavorite([id], !on);
      }}
      onKeyDown={(e) => {
        if (e.key === ' ' || e.key === 'Enter') {
          e.preventDefault();
          e.stopPropagation();
          useApp.getState().setFavorite([id], !on);
        }
      }}
    >
      <Icon name={on ? 'starFill' : 'star'} />
    </span>
  );
}
