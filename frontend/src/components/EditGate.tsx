import {useCallback, useState} from 'react'
import {BackupCurrent} from '../../wailsjs/go/main/App'
import {EventsOff, EventsOn} from '../../wailsjs/runtime'
import {imaging, main} from '../../wailsjs/go/models'
import {TransferProgress, bytes, duration, percent, rate} from '../lib/format'
import {errText} from '../lib/hex'
import {useT} from '../i18n'
import {useEffect} from 'react'

interface Props {
    disk: main.DiskInfo
    /** Where a backup of this medium already went, or "" if none. */
    backupPath: string
    onEnable: () => void
    onCancel: () => void
    onBackedUp: () => void
}

/** Stands between the user and enabling edits.
 *
 * Editing a card is irreversible and the ext4 write path is the least proven
 * part of the stack, so the gate does not merely warn — it offers the backup
 * itself, which makes the safe route the shortest one.
 */
export default function EditGate({disk, backupPath, onEnable, onCancel, onBackedUp}: Props) {
    const t = useT()
    const [progress, setProgress] = useState<TransferProgress | null>(null)
    const [running, setRunning] = useState(false)
    const [result, setResult] = useState<imaging.Result | null>(null)
    const [error, setError] = useState<string | null>(null)
    const [acknowledged, setAcknowledged] = useState(false)

    useEffect(() => {
        EventsOn('transfer:progress', (p: TransferProgress) => setProgress(p))
        return () => EventsOff('transfer:progress')
    }, [])

    const backup = useCallback(async () => {
        setRunning(true)
        setError(null)
        setResult(null)
        try {
            const res = await BackupCurrent()
            // A null result means the user closed the save dialog.
            if (res) {
                setResult(res)
                onBackedUp()
            }
        } catch (e) {
            setError(errText(e))
        } finally {
            setRunning(false)
            setProgress(null)
        }
    }, [onBackedUp])

    const haveBackup = backupPath !== '' || result !== null
    const isImage = !disk.path.startsWith('/dev/') && !disk.path.startsWith('\\\\.\\')

    return (
        <div className="copy-body">
            <div className="card danger-card">
                <h3>{t.gate.heading}</h3>

                <p className="small">{t.gate.intro(disk.path, disk.sizeHuman)}</p>
                <p className="dim small">{t.gate.rationale(isImage)}</p>

                {haveBackup && (
                    <p className="banner ok">
                        {t.gate.haveBackup(
                            result?.path ?? backupPath,
                            result ? t.gate.verified(bytes(result.bytes)) : null,
                        )}
                    </p>
                )}

                {error && <p className="banner error">{error}</p>}

                {progress && (
                    <div className="progress">
                        <div className="bar">
                            <div className="fill" style={{width: `${percent(progress.bytes, progress.total)}%`}} />
                        </div>
                        <div className="progress-meta dim">
                            <span>{progress.phase === 'verifying' ? t.copy.phaseVerifying : t.copy.phaseCopying}</span>
                            <span>{bytes(progress.bytes)} / {bytes(progress.total)}</span>
                            <span>{rate(progress.bytesPerSecond)}</span>
                            <span>{t.copy.remaining(duration(progress.etaSeconds))}</span>
                        </div>
                    </div>
                )}

                <div className="actions">
                    <button className="primary" onClick={backup} disabled={running}>
                        {running ? t.gate.copying : haveBackup ? t.gate.makeAnother : t.gate.makeCopy}
                    </button>
                    <button className="ghost" onClick={onCancel} disabled={running}>
                        {t.gate.cancel}
                    </button>
                </div>

                <hr className="divider" />

                {/* Skipping is allowed — it is the user's medium — but it takes a
                    deliberate act rather than a click straight past the warning. */}
                <label className="confirm">
                    <input
                        type="checkbox"
                        checked={acknowledged}
                        onChange={e => setAcknowledged(e.target.checked)}
                        disabled={running}
                    />
                    <span>
                        {haveBackup ? t.gate.acknowledgeWithBackup : t.gate.acknowledgeWithout}
                    </span>
                </label>

                <div className="actions">
                    <button className="danger" onClick={onEnable} disabled={running || !acknowledged}>
                        {t.gate.enable}
                    </button>
                </div>
            </div>
        </div>
    )
}
