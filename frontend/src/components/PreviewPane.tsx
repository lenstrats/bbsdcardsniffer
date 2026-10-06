import {useMemo, useState} from 'react'
import {main, volume} from '../../wailsjs/go/models'
import {decodeBase64, decodeText, hexdump} from '../lib/hex'
import {useT} from '../i18n'

interface Props {
    entry: volume.Entry | null
    preview: main.Preview | null
    loading: boolean
    error: string | null
    onExport: () => void
    onSeek: (offset: number) => void
}

export default function PreviewPane({entry, preview, loading, error, onExport, onSeek}: Props) {
    const [mode, setMode] = useState<'auto' | 'hex' | 'text'>('auto')
    const t = useT()

    const bytes = useMemo(
        () => (preview ? decodeBase64(preview.data) : new Uint8Array()),
        [preview],
    )

    if (!entry) {
        return <section className="preview empty-pane"><p className="empty">{t.preview.pickFile}</p></section>
    }

    const showText = mode === 'text' || (mode === 'auto' && preview?.isText)
    const hasMore = preview ? preview.offset + preview.length < preview.total : false

    return (
        <section className="preview">
            <div className="panel-head">
                <h2 className="mono">{entry.path}</h2>
                <div className="controls">
                    <div className="segmented">
                        {(['auto', 'text', 'hex'] as const).map(m => (
                            <button key={m} className={mode === m ? 'active' : ''} onClick={() => setMode(m)}>
                                {m === 'auto' ? t.preview.modeAuto : m === 'text' ? t.preview.modeText : t.preview.modeHex}
                            </button>
                        ))}
                    </div>
                    <button className="ghost small" onClick={onExport}>{t.preview.save}</button>
                </div>
            </div>

            {error && <p className="error">{error}</p>}
            {loading && <p className="empty">{t.browser.loading}</p>}

            {!loading && !error && preview && (
                <>
                    <pre className="dump">{showText ? decodeText(bytes) : hexdump(bytes, preview.offset)}</pre>
                    <div className="pager">
                        <span className="dim">
                            {t.preview.range(
                                preview.offset.toLocaleString(),
                                (preview.offset + preview.length).toLocaleString(),
                                preview.totalHuman,
                            )}
                        </span>
                        <span className="spacer" />
                        <button
                            className="ghost small"
                            onClick={() => onSeek(Math.max(0, preview.offset - preview.length))}
                            disabled={preview.offset === 0}
                        >
                            {t.preview.previous}
                        </button>
                        <button
                            className="ghost small"
                            onClick={() => onSeek(preview.offset + preview.length)}
                            disabled={!hasMore}
                        >
                            {t.preview.next}
                        </button>
                    </div>
                </>
            )}
        </section>
    )
}
