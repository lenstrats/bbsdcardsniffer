import {device} from '../../wailsjs/go/models'
import {useT} from '../i18n'

interface Props {
    devices: device.Device[]
    selectedPath: string | null
    busy: boolean
    canUnmount: boolean
    onSelect: (d: device.Device) => void
    onUnmount: (d: device.Device) => void
    onRefresh: () => void
    onOpenImage: () => void
}

export default function DeviceList(props: Props) {
    const {devices, selectedPath, busy, canUnmount, onSelect, onUnmount, onRefresh, onOpenImage} = props
    const t = useT()

    return (
        <aside className="devices">
            <div className="panel-head">
                <h2>{t.devices.heading}</h2>
                <button className="ghost" onClick={onRefresh} disabled={busy}>{t.devices.refresh}</button>
            </div>

            {devices.length === 0 && !busy && (
                <p className="empty">{t.devices.none}</p>
            )}

            <ul className="device-list">
                {devices.map(d => (
                    <li key={d.path}>
                        <button
                            className={'device' + (d.path === selectedPath ? ' active' : '')}
                            onClick={() => onSelect(d)}
                            disabled={busy}
                        >
                            <span className="device-name">{d.name || d.id}</span>
                            <span className="device-meta">
                                <span>{d.sizeHuman}</span>
                                {d.bus && <span>{d.bus}</span>}
                                {d.removable && <span className="tag ok">{t.devices.removable}</span>}
                                {d.internal && <span className="tag warn">{t.devices.internal}</span>}
                            </span>
                            <span className="device-node">{d.node}</span>
                        </button>

                        {/* A mounted disk on macOS cannot be read raw, so offer the fix inline. */}
                        {d.mounted && canUnmount && (
                            <button className="ghost small" onClick={() => onUnmount(d)} disabled={busy}>
                                {t.devices.unmount(d.id)}
                            </button>
                        )}
                    </li>
                ))}
            </ul>

            <div className="panel-foot">
                {/* An image opens through exactly the same code path as a card,
                    which makes a dd dump inspectable without the card present. */}
                <button className="ghost" onClick={onOpenImage} disabled={busy}>{t.devices.openImage}</button>
            </div>
        </aside>
    )
}
