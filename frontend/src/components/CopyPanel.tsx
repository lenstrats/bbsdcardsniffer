import {useCallback, useEffect, useState} from 'react'
import {CancelTransfer, CloneDevice, WriteImage} from '../../wailsjs/go/main/App'
import {EventsOff, EventsOn} from '../../wailsjs/runtime'
import {device, imaging, main} from '../../wailsjs/go/models'
import {TransferProgress, bytes, duration, percent, rate} from '../lib/format'
import {errText} from '../lib/hex'
import {useT} from '../i18n'
import CloneSection from './CloneSection'
import WriteSection from './WriteSection'

interface Props {
    /** The medium that is open — a card or an image — and can be copied. */
    disk: main.DiskInfo | null
    /** The card highlighted in the device list, which is what a write targets. */
    selected: device.Device | null
    /** Called after a write, since the card's contents have changed. */
    onDeviceChanged: () => void
}

type Running = 'clone' | 'write' | null

export default function CopyPanel({disk, selected, onDeviceChanged}: Props) {
    const t = useT()
    const [running, setRunning] = useState<Running>(null)
    const [progress, setProgress] = useState<TransferProgress | null>(null)
    const [result, setResult] = useState<imaging.Result | null>(null)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        EventsOn('transfer:progress', (p: TransferProgress) => setProgress(p))
        return () => EventsOff('transfer:progress')
    }, [])

    // A different medium means the previous run's outcome no longer applies.
    useEffect(() => {
        setResult(null)
        setError(null)
    }, [disk?.path, selected?.path])

    const run = useCallback(async (kind: Exclude<Running, null>, start: () => Promise<imaging.Result | null>) => {
        setRunning(kind)
        setError(null)
        setResult(null)
        setProgress(null)
        try {
            const res = await start()
            // A null result means the user closed the file dialog.
            if (res) setResult(res)
        } catch (e) {
            setError(errText(e))
        } finally {
            setRunning(null)
            setProgress(null)
            if (kind === 'write') onDeviceChanged()
        }
    }, [onDeviceChanged])

    // Copying reads the medium that is open, which may be an image file.
    const startClone = useCallback((opts: {trim: boolean; compress: boolean; verify: boolean}) => {
        if (!disk) return
        run('clone', () => CloneDevice({device: disk.path, ...opts}))
    }, [run, disk])

    const startWrite = useCallback((opts: {image: string; verify: boolean}) => {
        if (!selected) return
        run('write', () => WriteImage({device: selected.path, ...opts}))
    }, [run, selected])

    if (!disk && !selected) {
        return <section className="copy"><p className="empty big">{t.copy.pickMedium}</p></section>
    }

    const busy = running !== null

    return (
        <section className="copy">
            <div className="copy-body">
                {disk && (
                    <p className="target">
                        <strong>{t.copy.opened}</strong>
                        <span className="dim mono"> {disk.path}</span>
                        <span className="dim"> · {disk.sizeHuman} · {disk.table}</span>
                    </p>
                )}

                {disk && <CloneSection disk={disk} busy={busy} onStart={startClone} />}

                {selected ? (
                    <WriteSection selected={selected} busy={busy} onStart={startWrite} />
                ) : (
                    <div className="card danger-card">
                        <h3>{t.write.heading}</h3>
                        <p className="banner warn">{t.write.needTarget}</p>
                    </div>
                )}

                {progress && (
                    <div className="progress">
                        <div className="bar">
                            <div className="fill" style={{width: `${percent(progress.bytes, progress.total)}%`}} />
                        </div>
                        <div className="progress-meta dim">
                            <span>
                                {progress.phase === 'verifying'
                                    ? t.copy.phaseVerifying
                                    : running === 'write'
                                      ? t.copy.phaseWriting
                                      : t.copy.phaseCopying}
                            </span>
                            <span>{bytes(progress.bytes)} / {bytes(progress.total)}</span>
                            <span>{rate(progress.bytesPerSecond)}</span>
                            <span>{t.copy.remaining(duration(progress.etaSeconds))}</span>
                        </div>
                        <div className="actions">
                            <button className="ghost small" onClick={() => CancelTransfer().catch(() => {})}>
                                {t.copy.cancel}
                            </button>
                        </div>
                    </div>
                )}

                {error && <p className="banner error">{error}</p>}

                {result && (
                    <div className="banner ok result">
                        <p>
                            {t.copy.result(
                                bytes(result.bytes),
                                result.path,
                                result.fileBytes !== result.bytes ? bytes(result.fileBytes) : null,
                                duration(result.seconds),
                                result.verified,
                            )}
                        </p>
                        <p className="mono hash">sha256 {result.sha256}</p>
                    </div>
                )}
            </div>
        </section>
    )
}
