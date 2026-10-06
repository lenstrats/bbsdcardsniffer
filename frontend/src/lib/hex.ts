// Helpers for the preview pane. The backend hands over base64 so the raw bytes
// survive the JSON bridge intact.

export function decodeBase64(b64: string): Uint8Array {
    const bin = atob(b64)
    const out = new Uint8Array(bin.length)
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
    return out
}

/** Renders bytes as a classic 16-column hex dump, offsets absolute in the file. */
export function hexdump(bytes: Uint8Array, baseOffset: number): string {
    const lines: string[] = []
    for (let i = 0; i < bytes.length; i += 16) {
        const row = bytes.subarray(i, i + 16)
        const offset = (baseOffset + i).toString(16).padStart(8, '0')

        let hex = ''
        for (let j = 0; j < 16; j++) {
            hex += j < row.length ? row[j].toString(16).padStart(2, '0') : '  '
            hex += j === 7 ? '  ' : ' '
        }

        let ascii = ''
        for (const b of row) ascii += b >= 0x20 && b < 0x7f ? String.fromCharCode(b) : '.'

        lines.push(`${offset}  ${hex} |${ascii}|`)
    }
    return lines.join('\n')
}

export function decodeText(bytes: Uint8Array): string {
    return new TextDecoder('utf-8', {fatal: false}).decode(bytes)
}

/** Splits a display path into the crumbs a breadcrumb bar needs. */
export function crumbs(path: string): {name: string; path: string}[] {
    const parts = path.split('/').filter(Boolean)
    const out = [{name: '/', path: '/'}]
    let acc = ''
    for (const part of parts) {
        acc += '/' + part
        out.push({name: part, path: acc})
    }
    return out
}

export function parentOf(path: string): string {
    const parts = path.split('/').filter(Boolean)
    parts.pop()
    return '/' + parts.join('/')
}

/** Wails rejects with a plain string; normalise whatever we get to one. */
export function errText(e: unknown): string {
    if (typeof e === 'string') return e
    if (e instanceof Error) return e.message
    return String(e)
}
