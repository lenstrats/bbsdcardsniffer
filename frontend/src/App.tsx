import {useCallback, useEffect, useState} from 'react'
import {
    Environment,
    ExportFile,
    ExportTree,
    ListDevices,
    ListDir,
    OpenDevice,
    OpenImage,
    PreviewFile,
    SetLocale,
    UnmountDisk,
} from '../wailsjs/go/main/App'
import {device, main, volume} from '../wailsjs/go/models'
import AssistantPanel from './components/AssistantPanel'
import CopyPanel from './components/CopyPanel'
import DeviceList from './components/DeviceList'
import FileBrowser from './components/FileBrowser'
import PreviewPane from './components/PreviewPane'
import VolumeTabs from './components/VolumeTabs'
import {fsMeta} from './lib/fs'
import {errText} from './lib/hex'
import {I18nContext, Locale, detectLocale, localeOrder, locales, storeLocale} from './i18n'

/** How much of a file we pull in per preview page. */
const PREVIEW_LEN = 4096

export default function App() {
    const [env, setEnv] = useState<main.Environment | null>(null)
    const [devices, setDevices] = useState<device.Device[]>([])
    const [devError, setDevError] = useState<string | null>(null)
    const [busy, setBusy] = useState(false)
    const [notice, setNotice] = useState<string | null>(null)

    const [disk, setDisk] = useState<main.DiskInfo | null>(null)
    const [openError, setOpenError] = useState<string | null>(null)
    const [activeVol, setActiveVol] = useState(0)

    const [cwd, setCwd] = useState('/')
    const [entries, setEntries] = useState<volume.Entry[]>([])
    const [listLoading, setListLoading] = useState(false)
    const [listError, setListError] = useState<string | null>(null)

    const [selected, setSelected] = useState<volume.Entry | null>(null)
    const [preview, setPreview] = useState<main.Preview | null>(null)
    const [prevLoading, setPrevLoading] = useState(false)
    const [prevError, setPrevError] = useState<string | null>(null)
    const [exporting, setExporting] = useState(false)

    // The device list is shared by both modes, so the picked device is tracked
    // separately from the browsing session, which may have failed to open.
    const [selectedDevice, setSelectedDevice] = useState<device.Device | null>(null)
    const [mode, setMode] = useState<'browse' | 'copy' | 'assistant'>('browse')
    const [locale, setLocale] = useState<Locale>(detectLocale)

    // App provides the catalogue, so it reads it directly rather than through
    // the context it is itself supplying.
    const t = locales[locale]

    const chooseLocale = useCallback((next: Locale) => {
        setLocale(next)
        storeLocale(next)
        // The backend phrases its own errors, so it has to follow the choice.
        SetLocale(next).catch(() => {})
    }, [])

    useEffect(() => {
        SetLocale(locale).catch(() => {})
        // Runs once: later changes go through chooseLocale.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    const refreshDevices = useCallback(async () => {
        setBusy(true)
        setDevError(null)
        try {
            setDevices(await ListDevices())
        } catch (e) {
            setDevError(errText(e))
        } finally {
            setBusy(false)
        }
    }, [])

    useEffect(() => {
        refreshDevices()
    }, [refreshDevices])

    /** Shared by the device list and the image picker: both yield a DiskInfo. */
    const adopt = useCallback(async (load: () => Promise<main.DiskInfo | null>) => {
        setBusy(true)
        setOpenError(null)
        setNotice(null)
        setDisk(null)
        setSelected(null)
        setPreview(null)
        setEntries([])
        try {
            const info = await load()
            if (!info) return
            setDisk(info)
            // Land on the first volume we can actually browse, rather than on
            // an EFI or swap partition the user did not come for.
            const first = info.volumes.find(v => v.supported)
            setActiveVol(first ? first.index : 0)
            setCwd('/')
        } catch (e) {
            setOpenError(errText(e))
        } finally {
            setBusy(false)
        }
    }, [])

    const selectDevice = useCallback(
        (d: device.Device) => {
            setSelectedDevice(d)
            return adopt(() => OpenDevice(d.path))
        },
        [adopt],
    )

    const openImage = useCallback(() => {
        // Clear the highlighted card: what is open is now an image, and leaving
        // the old selection standing would point the copy tab at the wrong
        // medium while the rest of the app shows the image.
        setSelectedDevice(null)
        return adopt(() => OpenImage())
    }, [adopt])

    useEffect(() => {
        Environment()
            .then(e => {
                setEnv(e)
                // A path given on the command line opens without a click.
                if (e.autoOpen) adopt(() => OpenDevice(e.autoOpen))
            })
            .catch(() => {})
    }, [adopt])

    const unmount = useCallback(async (d: device.Device) => {
        setBusy(true)
        setNotice(null)
        try {
            await UnmountDisk(d.id)
            setNotice(t.devices.unmounted(d.id))
            await refreshDevices()
        } catch (e) {
            setDevError(errText(e))
        } finally {
            setBusy(false)
        }
    }, [refreshDevices])

    const activeVolume = disk?.volumes.find(v => v.index === activeVol) ?? null

    // Reload the listing whenever the volume or the directory changes.
    useEffect(() => {
        if (!disk || !activeVolume?.supported) {
            setEntries([])
            return
        }
        let cancelled = false
        setListLoading(true)
        setListError(null)
        ListDir(activeVol, cwd)
            .then(list => {
                if (!cancelled) setEntries(list)
            })
            .catch(e => {
                if (!cancelled) {
                    setEntries([])
                    setListError(errText(e))
                }
            })
            .finally(() => {
                if (!cancelled) setListLoading(false)
            })
        return () => {
            cancelled = true
        }
    }, [disk, activeVol, activeVolume?.supported, cwd])

    const loadPreview = useCallback(async (entry: volume.Entry, offset: number) => {
        setPrevLoading(true)
        setPrevError(null)
        try {
            setPreview(await PreviewFile(activeVol, entry.path, offset, PREVIEW_LEN))
        } catch (e) {
            setPreview(null)
            setPrevError(errText(e))
        } finally {
            setPrevLoading(false)
        }
    }, [activeVol])

    const selectEntry = useCallback((entry: volume.Entry) => {
        setSelected(entry)
        loadPreview(entry, 0)
    }, [loadPreview])

    const navigate = useCallback((path: string) => {
        setCwd(path)
        setSelected(null)
        setPreview(null)
        setPrevError(null)
    }, [])

    const selectVolume = useCallback((index: number) => {
        setActiveVol(index)
        setCwd('/')
        setSelected(null)
        setPreview(null)
        setPrevError(null)
    }, [])

    const exportSelected = useCallback(async () => {
        if (!selected) return
        setNotice(null)
        try {
            const dest = await ExportFile(activeVol, selected.path)
            if (dest) setNotice(t.browser.savedAs(dest))
        } catch (e) {
            setPrevError(errText(e))
        }
    }, [activeVol, selected])

    const exportTree = useCallback(async () => {
        setExporting(true)
        setNotice(null)
        try {
            const res = await ExportTree(activeVol, cwd)
            if (res) setNotice(t.browser.exported(res.files, res.dest, res.skipped?.length ?? 0))
        } catch (e) {
            setListError(errText(e))
        } finally {
            setExporting(false)
        }
    }, [activeVol, cwd])

    const meta = activeVolume ? fsMeta(activeVolume.fs.kind, t) : null

    return (
        <I18nContext.Provider value={t}>
        <div className="app">
            <header className="titlebar">
                <h1>{t.app.title}</h1>
                <div className="segmented modes">
                    <button className={mode === 'browse' ? 'active' : ''} onClick={() => setMode('browse')}>{t.app.mode.browse}</button>
                    <button className={mode === 'copy' ? 'active' : ''} onClick={() => setMode('copy')}>{t.app.mode.copy}</button>
                    <button className={mode === 'assistant' ? 'active' : ''} onClick={() => setMode('assistant')}>{t.app.mode.assistant}</button>
                </div>
                <span className="spacer" />
                <div className="segmented langs">
                    {localeOrder.map(code => (
                        <button
                            key={code}
                            className={locale === code ? 'active' : ''}
                            onClick={() => chooseLocale(code)}
                            title={locales[code].localeName}
                        >
                            {code.toUpperCase()}
                        </button>
                    ))}
                </div>
                {disk && <span className="mono dim">{disk.path} · {disk.sizeHuman} · {disk.table}</span>}
            </header>

            {/* Raw device reads need elevated rights; say so before the user hits a bare errno. */}
            {env && !env.elevated && env.os !== 'windows' && (
                <p className="banner warn">{t.app.notElevated}</p>
            )}
            {devError && <p className="banner error">{devError}</p>}
            {openError && <p className="banner error">{openError}</p>}
            {notice && <p className="banner ok">{notice}</p>}

            <main className="layout">
                <DeviceList
                    devices={devices}
                    selectedPath={selectedDevice?.path ?? disk?.path ?? null}
                    busy={busy}
                    canUnmount={env?.canUnmount ?? false}
                    onSelect={selectDevice}
                    onUnmount={unmount}
                    onRefresh={refreshDevices}
                    onOpenImage={openImage}
                />

                <div className="content">
                    {mode === 'copy' && <CopyPanel disk={disk} selected={selectedDevice} onDeviceChanged={refreshDevices} />}

                    {mode === 'assistant' && (
                        <AssistantPanel
                            disk={disk}
                            volume={activeVolume?.supported ? activeVolume : null}
                            onDiskChanged={setDisk}
                        />
                    )}

                    {mode === 'browse' && !disk && <p className="empty big">{t.app.pickDisk}</p>}

                    {mode === 'browse' && disk && (
                        <>
                            <VolumeTabs volumes={disk.volumes} active={activeVol} onSelect={selectVolume} />

                            {activeVolume && !activeVolume.supported && (
                                <p className="banner warn">
                                    {meta?.label}: {meta?.note}
                                </p>
                            )}

                            {activeVolume?.supported && (
                                <div className="split">
                                    <FileBrowser
                                        cwd={cwd}
                                        entries={entries}
                                        loading={listLoading}
                                        error={listError}
                                        selectedPath={selected?.path ?? null}
                                        onNavigate={navigate}
                                        onSelect={selectEntry}
                                        onExportTree={exportTree}
                                        exporting={exporting}
                                    />
                                    <PreviewPane
                                        entry={selected}
                                        preview={preview}
                                        loading={prevLoading}
                                        error={prevError}
                                        onExport={exportSelected}
                                        onSeek={offset => selected && loadPreview(selected, offset)}
                                    />
                                </div>
                            )}
                        </>
                    )}
                </div>
            </main>
        </div>
        </I18nContext.Provider>
    )
}
