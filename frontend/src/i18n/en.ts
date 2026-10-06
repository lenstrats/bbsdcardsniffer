/** The English catalogue, and the source of truth for the message shape.
 *
 * Every other locale is typed as `Messages`, so a missing or misspelled key is
 * a compile error rather than a blank spot in the interface. Strings that vary
 * with data are functions, which keeps word order the translator's decision
 * instead of forcing every language into English sentence structure.
 */
export const en = {
    // Shown in the language picker, always in the language itself.
    localeName: 'English',

    app: {
        title: 'SD Card Sniffer',
        mode: {browse: 'Browse', copy: 'Copy', assistant: 'Assistant'},
        notElevated: 'Not running as root — raw disk access will fail. Start it with sudo.',
        pickDisk: 'Pick a disk on the left to see its volumes.',
    },

    devices: {
        heading: 'Devices',
        refresh: 'Refresh',
        none: 'No disks found.',
        removable: 'removable',
        internal: 'internal',
        unmount: (id: string) => `Unmount (${id})`,
        unmounted: (id: string) => `${id} has been unmounted.`,
        openImage: 'Open disk image…',
    },

    volume: {
        unnamed: (n: number) => `Volume ${n}`,
    },

    browser: {
        up: 'Up one level',
        exportDir: 'Export folder',
        exporting: 'Exporting…',
        loading: 'Loading…',
        emptyDir: 'Empty folder.',
        colName: 'Name',
        colSize: 'Size',
        colMode: 'Permissions',
        colModified: 'Modified',
        exported: (files: number, dest: string, skipped: number) =>
            skipped > 0
                ? `${files} files to ${dest}, ${skipped} skipped`
                : `${files} files to ${dest}`,
        savedAs: (dest: string) => `Saved as ${dest}`,
    },

    preview: {
        pickFile: 'Select a file to see its contents.',
        modeAuto: 'auto',
        modeText: 'text',
        modeHex: 'hex',
        save: 'Save as…',
        previous: 'Previous',
        next: 'Next',
        range: (from: string, to: string, total: string) => `byte ${from}–${to} of ${total}`,
    },

    copy: {
        pickMedium: 'Pick a card on the left, or open an image, to copy or write.',
        opened: 'Opened:',
        cloneHeadingCard: 'Copy card to image',
        cloneHeadingImage: 'Copy image to a new image',
        cloneIntroCard: 'Reads the card out to a file. The card itself is left unchanged.',
        cloneIntroImage: 'Reads this image out to a file. The original is left unchanged.',
        trim: 'Stop after the last partition',
        trimDetail: (end: string, total: string, saved: string | null) =>
            saved ? ` — ${end} instead of ${total}, saving ${saved}` : ` — ${end} instead of ${total}`,
        compress: 'Compress (gzip)',
        verify: 'Verify afterwards',
        verifyDetailImage: ' — reads the image back and compares the hash',
        verifyDetailCard: ' — reads the card back and compares the hash',
        gptWarning:
            'This medium uses GPT. A trimmed image lacks the backup partition table at the end of the disk; gdisk can rebuild it after writing back.',
        startCard: 'Copy card…',
        startImage: 'Copy image…',
        cancel: 'Cancel',
        phaseCopying: 'Copying',
        phaseWriting: 'Writing',
        phaseVerifying: 'Verifying',
        remaining: (t: string) => `${t} left`,
        result: (amount: string, dest: string, onDisk: string | null, elapsed: string, verified: boolean) =>
            `${amount} to ${dest}${onDisk ? ` (${onDisk} on disk)` : ''} in ${elapsed}.${verified ? ' Verified.' : ''}`,
    },

    write: {
        heading: 'Write image to card',
        intro: 'Overwrites the whole card. Everything on it will be gone.',
        notRemovable: (name: string) =>
            `${name} is not a removable disk. Writing to it is blocked — including through the API.`,
        needTarget:
            'Pick a card on the left as the target. Writing needs a removable device; an opened image can only be the source.',
        pick: 'Choose image…',
        pickAnother: 'Choose a different image…',
        confirm: (name: string, node: string, size: string) =>
            `I understand that ${name} (${node}, ${size}) will be erased completely.`,
        start: 'Write to card',

        // What is currently on the target card.
        contentsHeading: 'Currently on this card',
        contentsEmpty: 'This card holds no partition table and no recognisable filesystem. It looks empty.',
        contentsUnreadable: (reason: string) =>
            `The contents of this card could not be read (${reason}). Anything on it will still be erased.`,
        contentsChecking: 'Reading the card…',
        contentsTable: (table: string, count: number) =>
            count === 1
                ? `${table.toUpperCase()} partition table, 1 volume:`
                : `${table.toUpperCase()} partition table, ${count} volumes:`,
        contentsNoTable: (count: number) =>
            count === 1 ? 'No partition table, 1 volume:' : `No partition table, ${count} volumes:`,
        contentsRow: (label: string, fs: string, size: string) => `${label} — ${fs}, ${size}`,
        unnamedVolume: 'unnamed',
        notEmptyWarning: 'This card is not empty. Writing destroys everything shown above.',
        confirmNotEmpty: (name: string, what: string) =>
            `I understand that ${name} will be erased, including ${what}.`,
        andMore: (n: number) => `and ${n} more`,
        tooLarge: (needed: string, capacity: string) =>
            `This image writes ${needed}, but the card holds only ${capacity}. It will not fit.`,
        sizeUnknown:
            'How much this image writes is only known while decompressing, so it is checked during the write.',
        writes: (amount: string) => `writes ${amount}`,
        format: {
            raw: 'uncompressed',
            gzip: 'gzip',
            xz: 'xz',
            bzip2: 'bzip2',
            zip: 'zip',
        } as Record<string, string>,
    },

    assistant: {
        pickVolume: 'Pick a disk and a readable volume to use the assistant.',
        heading: (label: string, fs: string) => `Assistant — ${label} (${fs})`,
        allowEdits: 'Allow edits',
        allowEditsHint: 'Reopens the disk so the assistant can change files',
        newChat: 'New conversation',
        writableBanner: 'The assistant may change files on this medium. There is no undo.',
        backupIs: (path: string) => ` Backup: ${path}`,
        noBackup: ' No backup was made this session.',
        promptSuggestions: 'Click a question to start, or type your own:',
        // Typed as a fixed-length tuple rather than string[], so adding a
        // suggestion here without adding one to every other locale is a
        // compile error instead of a silently shorter list.
        suggestions: [
            'This card stopped working. Find out what went wrong.',
            'What is running on this — which distribution, kernel and main services?',
            'Read the end of the logs in /var/log and summarise the latest errors.',
            'Why did the network not come up? Check the logs and the network configuration.',
            'Show me /etc/fstab and explain what gets mounted.',
        ] as [string, string, string, string, string],
        placeholder: 'Ask something about the files on this volume…',
        ask: 'Ask',
        stop: 'Stop',
        working: 'working…',
        failed: 'failed',
        apiKeyHeading: 'Claude API key',
        apiKeyIntro:
            'The assistant talks to the Claude API and needs a key for that. Set ANTHROPIC_API_KEY in your environment, or enter it here — it is then stored in your user configuration, readable only by you.',
        apiKeySave: 'Save',
    },

    gate: {
        heading: 'A backup first',
        intro: (path: string, size: string) =>
            `You are about to let the assistant write to ${path} (${size}). Changes are permanent — there is no undo.`,
        rationale: (isImage: boolean) =>
            `The ext4 write path in go-diskfs is far less proven than the read path, and we found four bugs in that read path. So make a full copy of ${isImage ? 'this image' : 'this card'} first. If something goes wrong, you simply write the copy back.`,
        haveBackup: (path: string, detail: string | null) =>
            `Backup present: ${path}${detail ? ` — ${detail}` : ''}`,
        verified: (amount: string) => `${amount}, verified`,
        makeCopy: 'Make a copy now…',
        makeAnother: 'Make another copy…',
        copying: 'Copying…',
        cancel: 'Cancel',
        acknowledgeWithBackup: 'I have a backup and want to continue.',
        acknowledgeWithout: 'I already have a backup elsewhere, or I can afford to lose this medium.',
        enable: 'Enable editing',
    },

    fs: {
        unknown: 'unknown',
        noDriver: (name: string) => `Recognised, but there is no read driver for ${name}.`,
        noSignature:
            'No known filesystem signature found — empty, damaged, or an unrecognised format.',
        swap: 'Swap space — there are no files on it.',
        luks: 'Encrypted. This app cannot unlock it; use cryptsetup.',
        lvm: 'LVM container. The real volumes live inside it and are not readable this way.',
    },
}

/** The shape every locale must provide. */
export type Messages = typeof en
