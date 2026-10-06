/** A transfer progress update, as emitted on the "transfer:progress" event.
 *
 * Wails only generates models for types that appear in a bound method's
 * signature. Progress only ever travels over the event bus, so its shape is
 * mirrored here by hand — keep it in step with imaging.Progress in Go.
 */
export interface TransferProgress {
    phase: 'copying' | 'verifying'
    bytes: number
    total: number
    bytesPerSecond: number
    /** Negative when there is not enough history to estimate. */
    etaSeconds: number
}

// Formatting helpers for transfer progress. Sizes here are decimal (MB/s) the
// way transfer rates are normally quoted, while capacities elsewhere use the
// binary units the Go side produces.

export function bytes(n: number): string {
    if (n < 1024) return `${n} B`
    const units = ['KiB', 'MiB', 'GiB', 'TiB']
    let v = n / 1024
    let i = 0
    while (v >= 1024 && i < units.length - 1) {
        v /= 1024
        i++
    }
    return `${v.toFixed(1)} ${units[i]}`
}

export function rate(bytesPerSecond: number): string {
    if (!isFinite(bytesPerSecond) || bytesPerSecond <= 0) return '—'
    return `${(bytesPerSecond / 1e6).toFixed(1)} MB/s`
}

/** Renders a duration as m:ss, or h:mm:ss once it runs past an hour. */
export function duration(seconds: number): string {
    if (!isFinite(seconds) || seconds < 0) return '—'
    const s = Math.round(seconds)
    const h = Math.floor(s / 3600)
    const m = Math.floor((s % 3600) / 60)
    const sec = s % 60
    if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
    return `${m}:${String(sec).padStart(2, '0')}`
}

export function percent(done: number, total: number): number {
    if (total <= 0) return 0
    return Math.min(100, (done / total) * 100)
}
