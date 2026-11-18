import { camName, decoderLabel, decoderShort, t } from '../../i18n/he';
import { urls } from '../../lib/api';
import { fmtBytes } from '../../lib/format';
import { engine } from '../../player/engine';
import { openClip } from '../../store/actions';
import { useApp } from '../../store/app';
import { CAMS, type Decoder } from '../../types';
import { StarButton } from '../common/StarButton';
import { Icon } from '../Icon';

const ORDER: Decoder[] = ['auto', 'native', 'compat'];

export function FilesPanel() {
  const detail = useApp((s) => s.current?.detail);
  const id = useApp((s) => s.current?.id);
  const ffmpeg = useApp((s) => !!s.status?.ffmpeg);
  const decoder = useApp((s) => s.decoder);
  if (!detail || !id) return null;

  const cycleDecoder = () => {
    const next = ORDER[(ORDER.indexOf(decoder) + 1) % ORDER.length];
    const st = useApp.getState();
    st.setDecoder(next);
    engine.decoder = next;
    st.toast(t.decoderSet(next));
    openClip(id, { t: engine.now(), play: engine.playing });
  };

  return (
    <div className="files">
      <div className="files-head">
        <span className="eyebrow">{t.files}</span>
        <code dir="ltr">{detail.clip.stem}</code>
        <StarButton id={id} />
      </div>
      {CAMS.map((c) => {
        const i = detail.cams[c];
        if (!i)
          return (
            <div key={c} className={`file off cam-${c}`}>
              <span className="dot" />
              <div>
                <b>{camName[c]}</b>
                <small>{t.notRecorded(c)}</small>
              </div>
            </div>
          );
        const spec = [i.width && `${i.width}×${i.height}`, i.fps && `${Math.round(i.fps)} fps`, i.codec?.toUpperCase(), fmtBytes(i.size)].filter(Boolean).join(' · ');
        return (
          <div key={c} className={`file cam-${c}`}>
            <span className="dot" />
            <div>
              <b>{camName[c]}</b>
              <small dir="ltr">{spec}</small>
              <small className="fname" dir="ltr">
                {i.file}
              </small>
            </div>
            <a className="mini" href={urls.download(id, c)} download title={t.downloadCam(c)}>
              <Icon name="download" />
            </a>
          </div>
        );
      })}
      {ffmpeg && (
        <div className="file decoder">
          <span className="dot" />
          <div>
            <b>{t.decoder}</b>
            <small>{decoderLabel[decoder]}</small>
          </div>
          <button className="mini" onClick={cycleDecoder}>
            {decoderShort[decoder]}
          </button>
        </div>
      )}
    </div>
  );
}
