import { useState } from 'react';
import { t } from '../../i18n/he';
import { disconnectMedia } from '../../store/actions';
import { useApp } from '../../store/app';
import { Modal } from '../common/Modal';
import { Icon } from '../Icon';

/** Confirm "disconnect media", optionally asking the OS to eject the drive. */
export function DisconnectModal() {
  const status = useApp((s) => s.status);
  const removable = !!status?.removable;
  const [eject, setEject] = useState(removable);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const go = async () => {
    setBusy(true);
    setError('');
    try {
      await disconnectMedia(eject && removable);
    } catch (e) {
      setError((e as Error).message);
      setBusy(false);
    }
  };

  return (
    <Modal
      title={t.disconnectTitle}
      className="disconnect-card"
      footer={
        <>
          <span />
          <div className="row">
            <button className="btn ghost" onClick={() => useApp.getState().openModal(null)}>
              {t.cancel}
            </button>
            <button className="btn primary" disabled={busy} onClick={go} autoFocus>
              <Icon name="eject" />
              {busy ? t.disconnecting : eject && removable ? t.disconnectEjectGo : t.disconnectGo}
            </button>
          </div>
        </>
      }
    >
      <div className="disconnect-body">
        <div className="media-chip">
          <Icon name="card" />
          <div>
            <b dir="ltr">{status?.label || status?.root}</b>
            {status?.label && <small dir="ltr">{status.root}</small>}
          </div>
        </div>
        <p className="muted">{t.disconnectBody}</p>
        {removable ? (
          <label className="opt">
            <input type="checkbox" checked={eject} onChange={(e) => setEject(e.target.checked)} />
            <span>
              <b>{t.disconnectEject}</b>
              <small>{t.disconnectEjectHint}</small>
            </span>
          </label>
        ) : (
          <p className="note">{t.disconnectNotRemovable}</p>
        )}
        {error && <p className="error">{error}</p>}
      </div>
    </Modal>
  );
}
