import {Messages} from '../i18n'

// Display metadata for the filesystem kinds the Go probe can report. The
// backend stays language-neutral: it returns a kind, and the wording is chosen
// here from the active catalogue.

export type Tone = 'ok' | 'warn' | 'info'

export interface FsMeta {
    label: string
    tone: Tone
    /** Shown when the volume cannot be browsed, to explain why. */
    note?: string
}

/** The display label and tone per kind. Labels are product names and stay the
 * same in every language; only the explanations are translated. */
const KINDS: Record<string, {label: string; tone: Tone; note?: (t: Messages) => string}> = {
    ext2: {label: 'ext2', tone: 'ok'},
    ext3: {label: 'ext3', tone: 'ok'},
    ext4: {label: 'ext4', tone: 'ok'},
    fat12: {label: 'FAT12', tone: 'ok'},
    fat16: {label: 'FAT16', tone: 'ok'},
    fat32: {label: 'FAT32', tone: 'ok'},
    iso9660: {label: 'ISO 9660', tone: 'ok'},
    squashfs: {label: 'SquashFS', tone: 'ok'},

    btrfs: {label: 'btrfs', tone: 'warn', note: t => t.fs.noDriver('btrfs')},
    xfs: {label: 'XFS', tone: 'warn', note: t => t.fs.noDriver('XFS')},
    f2fs: {label: 'F2FS', tone: 'warn', note: t => t.fs.noDriver('F2FS')},
    ntfs: {label: 'NTFS', tone: 'warn', note: t => t.fs.noDriver('NTFS')},
    exfat: {label: 'exFAT', tone: 'warn', note: t => t.fs.noDriver('exFAT')},
    apfs: {label: 'APFS', tone: 'warn', note: t => t.fs.noDriver('APFS')},
    hfsplus: {label: 'HFS+', tone: 'warn', note: t => t.fs.noDriver('HFS+')},

    swap: {label: 'Linux swap', tone: 'info', note: t => t.fs.swap},
    luks1: {label: 'LUKS1', tone: 'warn', note: t => t.fs.luks},
    luks2: {label: 'LUKS2', tone: 'warn', note: t => t.fs.luks},
    lvm2: {label: 'LVM2 PV', tone: 'warn', note: t => t.fs.lvm},
}

export function fsMeta(kind: string, t: Messages): FsMeta {
    const known = KINDS[kind]
    if (!known) {
        return {label: kind || t.fs.unknown, tone: 'warn', note: t.fs.noSignature}
    }
    return {label: known.label, tone: known.tone, note: known.note?.(t)}
}
