import {volume} from '../../wailsjs/go/models'
import {crumbs, parentOf} from '../lib/hex'
import {useT} from '../i18n'

interface Props {
    cwd: string
    entries: volume.Entry[]
    loading: boolean
    error: string | null
    selectedPath: string | null
    onNavigate: (path: string) => void
    onSelect: (entry: volume.Entry) => void
    onExportTree: () => void
    exporting: boolean
}

export default function FileBrowser(props: Props) {
    const {cwd, entries, loading, error, selectedPath, onNavigate, onSelect, onExportTree, exporting} = props
    const t = useT()

    // Directories first, then alphabetically — the order a file manager uses.
    const sorted = [...entries].sort((a, b) => {
        if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
        return a.name.localeCompare(b.name, 'nl')
    })

    return (
        <section className="browser">
            <div className="crumbs">
                <button className="ghost small" onClick={() => onNavigate(parentOf(cwd))} disabled={cwd === '/' || loading} title={t.browser.up}>
                    ↑
                </button>
                {crumbs(cwd).map((c, i) => (
                    <button key={c.path + i} className="crumb" onClick={() => onNavigate(c.path)} disabled={loading}>
                        {c.name}
                    </button>
                ))}
                <span className="spacer" />
                <button className="ghost small" onClick={onExportTree} disabled={loading || exporting}>
                    {exporting ? t.browser.exporting : t.browser.exportDir}
                </button>
            </div>

            {error && <p className="error">{error}</p>}
            {loading && <p className="empty">{t.browser.loading}</p>}
            {!loading && !error && sorted.length === 0 && <p className="empty">{t.browser.emptyDir}</p>}

            {!loading && !error && sorted.length > 0 && (
                <table className="files">
                    <thead>
                        <tr>
                            <th>{t.browser.colName}</th>
                            <th className="num">{t.browser.colSize}</th>
                            <th>{t.browser.colMode}</th>
                            <th>{t.browser.colModified}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {sorted.map(e => (
                            <tr
                                key={e.path}
                                className={e.path === selectedPath ? 'selected' : ''}
                                onClick={() => (e.isDir ? onNavigate(e.path) : onSelect(e))}
                            >
                                <td className="name">
                                    <span className="icon">{e.isDir ? '📁' : e.isSymlink ? '🔗' : '📄'}</span>
                                    {e.name}
                                </td>
                                <td className="num">{e.isDir ? '' : e.sizeHuman}</td>
                                <td className="mono dim">{e.mode}</td>
                                <td className="dim">{e.modTime ? e.modTime.replace('T', ' ').replace('Z', '') : ''}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </section>
    )
}
