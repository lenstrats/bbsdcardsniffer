import {useCallback, useEffect, useRef, useState} from 'react'
import {
    AskAssistant,
    AssistantStatus,
    CancelAssistant,
    EnableWrites,
    ResetAssistant,
    SetAPIKey,
} from '../../wailsjs/go/main/App'
import {EventsOff, EventsOn} from '../../wailsjs/runtime'
import {main, volume} from '../../wailsjs/go/models'
import {AgentEvent, Item, applyEvent, describeCall} from '../lib/assistant'
import {errText} from '../lib/hex'
import EditGate from './EditGate'
import {useT} from '../i18n'

interface Props {
    disk: main.DiskInfo | null
    volume: volume.Volume | null
    /** Called after the writable mode changes, so the rest of the UI keeps up. */
    onDiskChanged: (info: main.DiskInfo) => void
}

export default function AssistantPanel({disk, volume, onDiskChanged}: Props) {
    const t = useT()
    const [status, setStatus] = useState<main.AssistantStatus | null>(null)
    const [items, setItems] = useState<Item[]>([])
    const [prompt, setPrompt] = useState('')
    const [busy, setBusy] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [keyDraft, setKeyDraft] = useState('')
    // Set while the user is being asked to back up before edits are enabled.
    const [gateOpen, setGateOpen] = useState(false)
    const bottom = useRef<HTMLDivElement>(null)

    const refreshStatus = useCallback(() => {
        AssistantStatus().then(setStatus).catch(e => setError(errText(e)))
    }, [])

    useEffect(refreshStatus, [refreshStatus, disk])

    useEffect(() => {
        EventsOn('assistant:event', (e: AgentEvent) => setItems(prev => applyEvent(prev, e)))
        return () => EventsOff('assistant:event')
    }, [])

    // Switching volumes starts a new conversation on the Go side, so the
    // transcript has to follow.
    useEffect(() => {
        setItems([])
        setError(null)
    }, [volume?.index, disk?.path])

    useEffect(() => {
        bottom.current?.scrollIntoView({behavior: 'smooth'})
    }, [items])

    const send = useCallback(async () => {
        const text = prompt.trim()
        if (!text || !volume || busy) return
        setPrompt('')
        setError(null)
        setItems(prev => [...prev, {kind: 'user', text}])
        setBusy(true)
        try {
            await AskAssistant(volume.index, text)
        } catch (e) {
            setError(errText(e))
        } finally {
            setBusy(false)
            refreshStatus()
        }
    }, [prompt, volume, busy, refreshStatus])

    const applyWrites = useCallback(async (enable: boolean) => {
        setError(null)
        try {
            const info = await EnableWrites(enable)
            onDiskChanged(info)
            setItems([])
            refreshStatus()
        } catch (e) {
            setError(errText(e))
        }
    }, [onDiskChanged, refreshStatus])

    const toggleWrites = useCallback((enable: boolean) => {
        // Turning editing off is always safe and immediate; turning it on goes
        // through the backup gate.
        if (!enable) {
            setGateOpen(false)
            applyWrites(false)
            return
        }
        setGateOpen(true)
    }, [applyWrites])

    const saveKey = useCallback(async () => {
        try {
            await SetAPIKey(keyDraft.trim())
            setKeyDraft('')
            refreshStatus()
        } catch (e) {
            setError(errText(e))
        }
    }, [keyDraft, refreshStatus])

    if (!disk || !volume) {
        return <section className="assistant"><p className="empty big">{t.assistant.pickVolume}</p></section>
    }

    if (status && !status.hasKey) {
        return (
            <section className="assistant">
                <div className="copy-body">
                    <div className="card">
                        <h3>{t.assistant.apiKeyHeading}</h3>
                        <p className="dim small">{t.assistant.apiKeyIntro}</p>
                        <input
                            className="text-input"
                            type="password"
                            placeholder="sk-ant-…"
                            value={keyDraft}
                            onChange={e => setKeyDraft(e.target.value)}
                        />
                        <div className="actions">
                            <button className="primary" onClick={saveKey} disabled={!keyDraft.trim()}>{t.assistant.apiKeySave}</button>
                        </div>
                        {error && <p className="banner error">{error}</p>}
                    </div>
                </div>
            </section>
        )
    }

    return (
        <section className="assistant">
            <div className="panel-head">
                <h2>{t.assistant.heading(volume.label || t.volume.unnamed(volume.index + 1), volume.fs.kind)}</h2>
                <div className="controls">
                    <label className="inline-toggle" title={t.assistant.allowEditsHint}>
                        <input
                            type="checkbox"
                            checked={disk.writable}
                            onChange={e => toggleWrites(e.target.checked)}
                            disabled={busy}
                        />
                        <span>{t.assistant.allowEdits}</span>
                    </label>
                    <button className="ghost small" onClick={() => { ResetAssistant(); setItems([]) }} disabled={busy}>
                        {t.assistant.newChat}
                    </button>
                </div>
            </div>

            {gateOpen && !disk.writable && (
                <EditGate
                    disk={disk}
                    backupPath={status?.backupPath ?? ''}
                    onEnable={() => { setGateOpen(false); applyWrites(true) }}
                    onCancel={() => setGateOpen(false)}
                    onBackedUp={refreshStatus}
                />
            )}

            {disk.writable && (
                <p className="banner warn">
                    {t.assistant.writableBanner}
                    {status?.backupPath ? t.assistant.backupIs(status.backupPath) : t.assistant.noBackup}
                </p>
            )}

            {!gateOpen && <div className="transcript">
                {items.length === 0 && (
                    <div className="suggestions">
                        <p className="dim">{t.assistant.promptSuggestions}</p>
                        <ul>
                            {t.assistant.suggestions.map(q => (
                                <li key={q}>
                                    <button className="suggestion" onClick={() => setPrompt(q)}>{q}</button>
                                </li>
                            ))}
                        </ul>
                    </div>
                )}

                {items.map((item, i) => {
                    if (item.kind === 'user') {
                        return <div key={i} className="bubble user">{item.text}</div>
                    }
                    if (item.kind === 'assistant') {
                        return <div key={i} className="bubble agent">{item.text}</div>
                    }
                    return (
                        <details key={i} className={'tool-call' + (item.mutating ? ' mutating' : '') + (item.failed ? ' failed' : '')}>
                            <summary>
                                <span className="mono">{describeCall(item.tool, item.input)}</span>
                                {item.result === undefined && <span className="dim"> — {t.assistant.working}</span>}
                                {item.failed && <span className="dim"> — {t.assistant.failed}</span>}
                            </summary>
                            <pre>{item.result ?? ''}</pre>
                        </details>
                    )
                })}
                <div ref={bottom} />
            </div>}

            {error && <p className="banner error">{error}</p>}

            {!gateOpen && <div className="composer">
                <textarea
                    value={prompt}
                    onChange={e => setPrompt(e.target.value)}
                    onKeyDown={e => {
                        // Enter sends; Shift+Enter makes a new line.
                        if (e.key === 'Enter' && !e.shiftKey) {
                            e.preventDefault()
                            send()
                        }
                    }}
                    placeholder={t.assistant.placeholder}
                    rows={3}
                    disabled={busy}
                />
                <div className="actions">
                    {busy ? (
                        <button className="ghost" onClick={() => CancelAssistant().catch(() => {})}>{t.assistant.stop}</button>
                    ) : (
                        <button className="primary" onClick={send} disabled={!prompt.trim()}>{t.assistant.ask}</button>
                    )}
                </div>
            </div>}
        </section>
    )
}
