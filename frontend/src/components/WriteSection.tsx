import {useEffect, useState} from 'react'
import {ChooseImage, InspectTarget} from '../../wailsjs/go/main/App'
import {device, main} from '../../wailsjs/go/models'
import {errText} from '../lib/hex'
import {useT} from '../i18n'
import {fsMeta} from '../lib/fs'

interface Props {
    selected: device.Device
    busy: boolean
    onStart: (opts: {image: string; verify: boolean}) => void
}

export default function WriteSection({selected, busy, onStart}: Props) {
    const t = useT()
    const [image, setImage] = useState<main.ImageInfo | null>(null)
    const [target, setTarget] = useState<main.TargetInfo | null>(null)
    const [verify, setVerify] = useState(true)
    const [confirmed, setConfirmed] = useState(false)
    const [error, setError] = useState<string | null>(null)

    // Picking another card must retract the confirmation, or a tick meant for
    // one disk would carry over to the next.
    useEffect(() => {
        setConfirmed(false)
        setTarget(null)
        if (!selected.removable || selected.internal) return

        let cancelled = false
        InspectTarget(selected.path)
            .then(info => !cancelled && setTarget(info))
            .catch(() => {
                // Never leave the user believing an unreadable card is blank.
                if (!cancelled) {
                    // createFrom is what the generated model expects; a plain
                    // literal is missing its conversion helpers.
                    setTarget(main.TargetInfo.createFrom({
                        path: selected.path,
                        node: selected.node,
                        size: selected.size,
                        sizeHuman: selected.sizeHuman,
                        table: '',
                        volumes: [],
                        empty: false,
                        unreadable: true,
                        detail: '',
                    }))
                }
            })
        return () => {
            cancelled = true
        }
    }, [selected])

    // Writing is refused in the backend for anything fixed; say so up front
    // rather than letting the user get as far as pressing the button.
    if (!selected.removable || selected.internal) {
        return (
            <div className="card danger-card">
                <h3>{t.write.heading}</h3>
                <p className="banner warn">{t.write.notRemovable(selected.name || selected.id)}</p>
            </div>
        )
    }

    const pick = async () => {
        setError(null)
        try {
            const picked = await ChooseImage()
            if (picked) {
                setImage(picked)
                setConfirmed(false)
            }
        } catch (e) {
            setError(errText(e))
        }
    }

    // Only refuse up front when the size is actually known: for xz and gzip it
    // is not, and the check then happens during the write instead.
    const tooLarge = !!image && image.written > 0 && selected.size > 0 && image.written > selected.size

    // A short list of what the card holds, for the confirmation sentence.
    const named = target && !target.unreadable && !target.empty
        ? target.volumes.map(v => v.label || fsMeta(v.fs.kind, t).label).filter(Boolean)
        : []
    const whatIsLost = named.length === 0
        ? null
        : named.length <= 3
          ? named.join(', ')
          : `${named.slice(0, 3).join(', ')} ${t.write.andMore(named.length - 3)}`

    return (
        <div className="card danger-card">
            <h3>{t.write.heading}</h3>
            <p className="dim small">{t.write.intro}</p>

            {error && <p className="banner error">{error}</p>}

            <div className="actions">
                <button className="ghost" onClick={pick} disabled={busy}>
                    {image ? t.write.pickAnother : t.write.pick}
                </button>
            </div>

            {image && (
                <p className="picked">
                    <strong>{image.name}</strong>
                    <span className="dim">
                        {' '}· {image.sizeHuman} · {t.write.format[image.format] ?? image.format}
                        {image.writtenHuman
                            ? ` · ${t.write.writes(image.writtenHuman)}`
                            : image.format !== 'raw' && ''}
                    </span>
                </p>
            )}

            {/* What the card holds now, so the confirmation names what is lost. */}
            <div className="contents">
                <h4>{t.write.contentsHeading}</h4>
                {!target && <p className="dim small">{t.write.contentsChecking}</p>}
                {target?.unreadable && (
                    <p className="banner warn">{t.write.contentsUnreadable(target.detail || '—')}</p>
                )}
                {target && !target.unreadable && target.empty && (
                    <p className="dim small">{t.write.contentsEmpty}</p>
                )}
                {target && !target.unreadable && !target.empty && (
                    <>
                        <p className="dim small">
                            {target.table && target.table !== 'none'
                                ? t.write.contentsTable(target.table, target.volumes.length)
                                : t.write.contentsNoTable(target.volumes.length)}
                        </p>
                        <ul className="volume-list">
                            {target.volumes.map(v => (
                                <li key={v.index}>
                                    {t.write.contentsRow(
                                        v.label || t.write.unnamedVolume,
                                        fsMeta(v.fs.kind, t).label,
                                        v.sizeHuman,
                                    )}
                                </li>
                            ))}
                        </ul>
                        <p className="banner warn">{t.write.notEmptyWarning}</p>
                    </>
                )}
            </div>

            {tooLarge && (
                <p className="banner error">
                    {t.write.tooLarge(image!.writtenHuman, selected.sizeHuman)}
                </p>
            )}

            <div className="options">
                <label>
                    <input type="checkbox" checked={verify} onChange={e => setVerify(e.target.checked)} disabled={busy} />
                    <span>
                        {t.copy.verify}
                        <span className="dim">{t.copy.verifyDetailCard}</span>
                    </span>
                </label>

                <label className="confirm">
                    <input
                        type="checkbox"
                        checked={confirmed}
                        onChange={e => setConfirmed(e.target.checked)}
                        disabled={busy || !image || tooLarge}
                    />
                    <span>
                        {whatIsLost
                            ? t.write.confirmNotEmpty(`${selected.name || selected.id} (${selected.node}, ${selected.sizeHuman})`, whatIsLost)
                            : t.write.confirm(selected.name || selected.id, selected.node, selected.sizeHuman)}
                    </span>
                </label>
            </div>

            <div className="actions">
                <button
                    className="danger"
                    onClick={() => image && onStart({image: image.path, verify})}
                    disabled={busy || !image || !confirmed || tooLarge}
                >
                    {t.write.start}
                </button>
            </div>
        </div>
    )
}
