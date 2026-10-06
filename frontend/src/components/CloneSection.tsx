import {useEffect, useState} from 'react'
import {InspectForClone} from '../../wailsjs/go/main/App'
import {main} from '../../wailsjs/go/models'
import {bytes} from '../lib/format'
import {errText} from '../lib/hex'
import {useT} from '../i18n'

interface Props {
    /** The medium that is open: a card or an image file. Both can be copied. */
    disk: main.DiskInfo
    busy: boolean
    onStart: (opts: {trim: boolean; compress: boolean; verify: boolean}) => void
}

export default function CloneSection({disk, busy, onStart}: Props) {
    const isImage = !disk.path.startsWith('/dev/') && !disk.path.startsWith('\\\\.\\')
    const t = useT()
    const [info, setInfo] = useState<main.TrimInfo | null>(null)
    const [error, setError] = useState<string | null>(null)

    const [trim, setTrim] = useState(true)
    const [compress, setCompress] = useState(false)
    const [verify, setVerify] = useState(true)

    // Inspect the card so the trim choice shows what it actually saves.
    useEffect(() => {
        setInfo(null)
        setError(null)
        let cancelled = false
        InspectForClone(disk.path)
            .then(i => !cancelled && setInfo(i))
            .catch(e => !cancelled && setError(errText(e)))
        return () => {
            cancelled = true
        }
    }, [disk.path])

    const saved = info && info.dataEnd < info.total ? info.total - info.dataEnd : 0

    return (
        <div className="card">
            <h3>{isImage ? t.copy.cloneHeadingImage : t.copy.cloneHeadingCard}</h3>
            <p className="dim small">{isImage ? t.copy.cloneIntroImage : t.copy.cloneIntroCard}</p>

            {error && <p className="banner error">{error}</p>}

            <div className="options">
                <label>
                    <input type="checkbox" checked={trim} onChange={e => setTrim(e.target.checked)} disabled={busy || !info} />
                    <span>
                        {t.copy.trim}
                        {info && (
                            <span className="dim">
                                {t.copy.trimDetail(info.dataEndHuman, info.totalHuman, saved > 0 ? bytes(saved) : null)}
                            </span>
                        )}
                    </span>
                </label>

                <label>
                    <input type="checkbox" checked={compress} onChange={e => setCompress(e.target.checked)} disabled={busy} />
                    <span>{t.copy.compress}</span>
                </label>

                <label>
                    <input type="checkbox" checked={verify} onChange={e => setVerify(e.target.checked)} disabled={busy} />
                    <span>
                        {t.copy.verify}
                        <span className="dim">{t.copy.verifyDetailImage}</span>
                    </span>
                </label>
            </div>

            {/* GPT keeps a backup table in the last sectors, which a trimmed
                image cannot contain. Say so rather than trim silently. */}
            {trim && info?.losesBackupGpt && (
                <p className="banner warn">{t.copy.gptWarning}</p>
            )}

            <div className="actions">
                <button className="primary" onClick={() => onStart({trim, compress, verify})} disabled={busy || !info}>
                    {isImage ? t.copy.startImage : t.copy.startCard}
                </button>
            </div>
        </div>
    )
}
