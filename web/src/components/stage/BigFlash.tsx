import { useEffect, useState } from 'react';
import { Icon } from '../Icon';

type FlashIcon = 'play' | 'pause';
const EVENT = 'dm:flash';

/** Show a big play/pause pulse over the stage. */
export const flash = (icon: FlashIcon) => window.dispatchEvent(new CustomEvent(EVENT, { detail: icon }));

export function BigFlash() {
  const [state, setState] = useState<{ icon: FlashIcon; n: number }>({ icon: 'play', n: 0 });
  useEffect(() => {
    const on = (e: Event) => setState((s) => ({ icon: (e as CustomEvent<FlashIcon>).detail, n: s.n + 1 }));
    window.addEventListener(EVENT, on);
    return () => window.removeEventListener(EVENT, on);
  }, []);
  if (!state.n) return null;
  return (
    <div key={state.n} className="big-flash go" aria-hidden="true">
      <Icon name={state.icon} />
    </div>
  );
}
