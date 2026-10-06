import {volume} from '../../wailsjs/go/models'
import {fsMeta} from '../lib/fs'
import {useT} from '../i18n'

interface Props {
    volumes: volume.Volume[]
    active: number
    onSelect: (index: number) => void
}

export default function VolumeTabs({volumes, active, onSelect}: Props) {
    const t = useT()

    return (
        <div className="tabs" role="tablist">
            {volumes.map(v => {
                const meta = fsMeta(v.fs.kind, t)
                const title = v.label || t.volume.unnamed(v.index + 1)
                return (
                    <button
                        key={v.index}
                        role="tab"
                        aria-selected={v.index === active}
                        className={'tab' + (v.index === active ? ' active' : '') + (v.supported ? '' : ' unsupported')}
                        onClick={() => onSelect(v.index)}
                        title={meta.note ?? ''}
                    >
                        <span className="tab-title">{title}</span>
                        <span className={'tag ' + meta.tone}>{meta.label}</span>
                        <span className="tab-size">{v.sizeHuman}</span>
                    </button>
                )
            })}
        </div>
    )
}
