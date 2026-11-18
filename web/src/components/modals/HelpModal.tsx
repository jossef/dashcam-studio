import { t } from '../../i18n/he';
import { Modal } from '../common/Modal';

export function HelpModal() {
  return (
    <Modal title={t.keyboard} className="help-card">
      <dl className="keys">
        {t.keys.map(([keys, what]) => (
          <div key={what}>
            <dt>
              {keys.map((k) => (
                <kbd key={k}>{k}</kbd>
              ))}
            </dt>
            <dd>{what}</dd>
          </div>
        ))}
      </dl>
      <p className="keys-note muted">{t.keysNote}</p>
    </Modal>
  );
}
