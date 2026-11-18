import { useEffect, useRef, useState } from 'react';
import { t } from '../../i18n/he';
import { api } from '../../lib/api';
import { storage } from '../../lib/storage';
import { openRoot } from '../../store/actions';
import { useApp } from '../../store/app';
import { Brand } from '../common/Brand';
import { Icon } from '../Icon';

export function WelcomeScreen() {
  const status = useApp((s) => s.status);
  const hasLibrary = useApp((s) => s.clips.length > 0);
  const [path, setPath] = useState(() => status?.root || storage.get<string[]>('recents', [])[0] || '');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(status?.error && !status.root ? status.error : '');
  const [drives, setDrives] = useState<{ path: string; name: string; fresh?: boolean }[]>([]);
  const notice = useApp((s) => s.notice);
  const recents = storage.get<string[]>('recents', []);
  const input = useRef<HTMLInputElement>(null);

  // keep looking for cards: one plugged in while this screen is open shows up by itself
  useEffect(() => {
    setTimeout(() => input.current?.focus(), 50);
    let first = true;
    let known = new Set<string>();
    const poll = () =>
      api
        .drives()
        .then((r) => {
          setDrives(r.drives.map((d) => ({ ...d, fresh: !first && !known.has(d.path) })));
          known = new Set(r.drives.map((d) => d.path));
          first = false;
        })
        .catch(() => {});
    poll();
    const id = window.setInterval(poll, 2500);
    return () => clearInterval(id);
  }, []);

  const submit = async (p: string) => {
    if (!p.trim()) return;
    setBusy(true);
    setError('');
    try {
      await openRoot(p.trim());
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  // the browse modal reports its pick through this event
  useEffect(() => {
    const onPick = (e: Event) => {
      const p = (e as CustomEvent<string>).detail;
      setPath(p);
      submit(p);
    };
    window.addEventListener('dm:pick-folder', onPick);
    return () => window.removeEventListener('dm:pick-folder', onPick);
  }, []);

  return (
    <section className="welcome">
      <div className="welcome-road-layer" aria-hidden="true">
        <div className="welcome-road">
          {Array.from({ length: 6 }, (_, i) => (
            <i key={i} />
          ))}
        </div>
      </div>
      <div className="welcome-card">
        <Brand large />
        <h1>{t.welcomeTitle}</h1>
        <p className="muted">{t.welcomeBody}</p>

        {notice && (
          <div className={`notice ${notice.kind}`} role="status">
            <Icon name={notice.kind === 'ok' ? 'check' : 'warn'} />
            <div>
              <b>{notice.title}</b>
              {notice.body && (
                <small>
                  <bdi>{notice.body}</bdi>
                </small>
              )}
            </div>
            <button className="icon-btn" aria-label="סגירה" onClick={() => useApp.getState().setNotice(null)}>
              <Icon name="x" />
            </button>
          </div>
        )}

        {drives.length === 0 && (
          <div className="drive-wait">
            <span className="pulse" />
            {t.waitingForCard}
          </div>
        )}
        {drives.length > 0 && (
          <div className="drives">
            {drives.map((d, i) => (
              <button key={d.path} type="button" className="drive" style={{ animationDelay: `${0.1 + i * 0.07}s` }} onClick={() => submit(d.path)}>
                <Icon name="drive" />
                <div>
                  <b dir="ltr">{d.name}</b>
                  {d.fresh && <em className="fresh">{t.newCard}</em>}
                  <small>
                    <span dir="ltr">{d.path}</span> · {t.cardDetected}
                  </small>
                </div>
                <span className="go">{t.open} ←</span>
              </button>
            ))}
          </div>
        )}

        <form
          className="open-form"
          autoComplete="off"
          onSubmit={(e) => {
            e.preventDefault();
            submit(path);
          }}
        >
          <label htmlFor="root-input" className="eyebrow">
            {t.rootFolder}
          </label>
          <div className="open-row">
            <input id="root-input" ref={input} dir="ltr" spellCheck={false} placeholder={t.rootPlaceholder} value={path} onChange={(e) => setPath(e.target.value)} />
            <button type="button" className="btn ghost" onClick={() => useApp.getState().openModal({ kind: 'browse', start: path.trim() })}>
              <Icon name="folder" />
              {t.browse}
            </button>
            <button type="submit" className="btn primary" disabled={busy}>
              {busy ? t.scanning : t.open}
            </button>
          </div>
          <p className="error" role="alert">
            {error}
          </p>
        </form>

        <div className="recents">
          {hasLibrary && (
            <button type="button" className="mini" onClick={() => useApp.getState().setWelcome(false)}>
              → {t.backToPlayer}
            </button>
          )}
          {recents.map((r) => (
            <button key={r} type="button" className="mini" dir="ltr" onClick={() => submit(r)}>
              {r}
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}
