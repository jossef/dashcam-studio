import { useApp } from '../../store/app';

export function Toasts() {
  const toasts = useApp((s) => s.toasts);
  return (
    <div className="toasts" aria-live="polite">
      {toasts.map((x) => (
        <div key={x.id} className={`toast ${x.kind ?? ''}`}>
          <span>{x.msg}</span>
          {x.sub && <small>{x.sub}</small>}
        </div>
      ))}
    </div>
  );
}
