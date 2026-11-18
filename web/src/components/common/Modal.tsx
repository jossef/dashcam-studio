import { useEffect, type ReactNode } from 'react';
import { useApp } from '../../store/app';
import { Icon } from '../Icon';

export function Modal({ title, children, footer, className = '' }: { title: ReactNode; children: ReactNode; footer?: ReactNode; className?: string }) {
  const close = () => useApp.getState().openModal(null);
  useEffect(() => {
    const prev = document.activeElement as HTMLElement | null;
    return () => prev?.focus?.();
  }, []);
  return (
    <div className="modal" role="dialog" aria-modal="true" onMouseDown={(e) => e.target === e.currentTarget && close()}>
      <div className={`modal-card ${className}`}>
        <header>
          <h2>{title}</h2>
          <button className="icon-btn" onClick={close} aria-label="סגירה">
            <Icon name="x" />
          </button>
        </header>
        {children}
        {footer && <footer>{footer}</footer>}
      </div>
    </div>
  );
}
