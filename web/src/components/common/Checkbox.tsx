import type { MouseEvent } from 'react';
import { Icon } from '../Icon';

/** Tri-state selection box used by clip rows, trips and days. */
export function Checkbox({
  state,
  onToggle,
  label,
  className = '',
}: {
  state: boolean | 'mixed';
  onToggle: (e: MouseEvent) => void;
  label: string;
  className?: string;
}) {
  return (
    <span
      role="checkbox"
      tabIndex={0}
      aria-checked={state === 'mixed' ? 'mixed' : state}
      aria-label={label}
      title={label}
      className={`check ${state ? 'on' : ''} ${className}`}
      onClick={(e) => {
        e.stopPropagation();
        e.preventDefault();
        onToggle(e);
      }}
      onKeyDown={(e) => {
        if (e.key === ' ' || e.key === 'Enter') {
          e.preventDefault();
          e.stopPropagation();
          onToggle(e as unknown as MouseEvent);
        }
      }}
    >
      {state === 'mixed' ? <Icon name="minus" /> : state ? <Icon name="check" /> : null}
    </span>
  );
}

export const selectionState = (ids: string[], selection: Set<string>): boolean | 'mixed' => {
  const n = ids.filter((id) => selection.has(id)).length;
  return n === 0 ? false : n === ids.length ? true : 'mixed';
};
