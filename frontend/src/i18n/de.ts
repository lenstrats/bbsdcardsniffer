import {Messages} from './en'

/** The German catalogue. Typed as Messages, so anything missing fails the build. */
export const de: Messages = {
    localeName: 'Deutsch',

    app: {
        title: 'SD Card Sniffer',
        mode: {browse: 'Durchsuchen', copy: 'Kopieren', assistant: 'Assistent'},
        notElevated: 'Nicht als root gestartet — der Rohzugriff auf Datenträger wird fehlschlagen. Mit sudo starten.',
        pickDisk: 'Links einen Datenträger wählen, um seine Volumes zu sehen.',
    },

    devices: {
        heading: 'Geräte',
        refresh: 'Aktualisieren',
        none: 'Keine Datenträger gefunden.',
        removable: 'wechselbar',
        internal: 'intern',
        unmount: (id: string) => `Aushängen (${id})`,
        unmounted: (id: string) => `${id} wurde ausgehängt.`,
        openImage: 'Image öffnen…',
    },

    volume: {
        unnamed: (n: number) => `Volume ${n}`,
    },

    browser: {
        up: 'Eine Ebene höher',
        exportDir: 'Ordner exportieren',
        exporting: 'Wird exportiert…',
        loading: 'Wird geladen…',
        emptyDir: 'Leerer Ordner.',
        colName: 'Name',
        colSize: 'Größe',
        colMode: 'Rechte',
        colModified: 'Geändert',
        exported: (files: number, dest: string, skipped: number) =>
            skipped > 0
                ? `${files} Dateien nach ${dest}, ${skipped} übersprungen`
                : `${files} Dateien nach ${dest}`,
        savedAs: (dest: string) => `Gespeichert als ${dest}`,
    },

    preview: {
        pickFile: 'Eine Datei auswählen, um ihren Inhalt zu sehen.',
        modeAuto: 'auto',
        modeText: 'Text',
        modeHex: 'Hex',
        save: 'Speichern unter…',
        previous: 'Zurück',
        next: 'Weiter',
        range: (from: string, to: string, total: string) => `Byte ${from}–${to} von ${total}`,
    },

    copy: {
        pickMedium: 'Links eine Karte wählen oder ein Image öffnen, um zu kopieren oder zu schreiben.',
        opened: 'Geöffnet:',
        cloneHeadingCard: 'Karte in ein Image kopieren',
        cloneHeadingImage: 'Image in ein neues Image kopieren',
        cloneIntroCard: 'Liest die Karte in eine Datei aus. Die Karte selbst bleibt unverändert.',
        cloneIntroImage: 'Liest dieses Image in eine Datei aus. Das Original bleibt unverändert.',
        trim: 'Nach der letzten Partition aufhören',
        trimDetail: (end: string, total: string, saved: string | null) =>
            saved ? ` — ${end} statt ${total}, spart ${saved}` : ` — ${end} statt ${total}`,
        compress: 'Komprimieren (gzip)',
        verify: 'Anschließend prüfen',
        verifyDetailImage: ' — liest das Image zurück und vergleicht die Prüfsumme',
        verifyDetailCard: ' — liest die Karte zurück und vergleicht die Prüfsumme',
        gptWarning:
            'Dieser Datenträger nutzt GPT. Einem abgeschnittenen Image fehlt die Sicherungs-Partitionstabelle am Ende des Datenträgers; gdisk kann sie nach dem Zurückschreiben wiederherstellen.',
        startCard: 'Karte kopieren…',
        startImage: 'Image kopieren…',
        cancel: 'Abbrechen',
        phaseCopying: 'Kopieren',
        phaseWriting: 'Schreiben',
        phaseVerifying: 'Prüfen',
        remaining: (t: string) => `noch ${t}`,
        result: (amount: string, dest: string, onDisk: string | null, elapsed: string, verified: boolean) =>
            `${amount} nach ${dest}${onDisk ? ` (${onDisk} auf der Platte)` : ''} in ${elapsed}.${verified ? ' Geprüft.' : ''}`,
    },

    write: {
        heading: 'Image auf Karte schreiben',
        intro: 'Überschreibt die gesamte Karte. Alles, was darauf ist, geht verloren.',
        notRemovable: (name: string) =>
            `${name} ist kein Wechseldatenträger. Das Schreiben darauf ist gesperrt — auch über die API.`,
        needTarget:
            'Links eine Karte als Ziel wählen. Zum Schreiben ist ein Wechseldatenträger nötig; ein geöffnetes Image kann nur die Quelle sein.',
        pick: 'Image wählen…',
        pickAnother: 'Anderes Image wählen…',
        confirm: (name: string, node: string, size: string) =>
            `Mir ist klar, dass ${name} (${node}, ${size}) vollständig gelöscht wird.`,
        start: 'Auf Karte schreiben',
        format: {
            raw: 'unkomprimiert',
            gzip: 'gzip',
            xz: 'xz',
            bzip2: 'bzip2',
            zip: 'zip',
        },

        contentsHeading: 'Derzeit auf dieser Karte',
        contentsEmpty: 'Diese Karte hat keine Partitionstabelle und kein erkennbares Dateisystem. Sie wirkt leer.',
        contentsUnreadable: (reason: string) =>
            `Der Inhalt dieser Karte konnte nicht gelesen werden (${reason}). Was darauf ist, wird trotzdem gelöscht.`,
        contentsChecking: 'Karte wird gelesen…',
        contentsTable: (table: string, count: number) =>
            count === 1
                ? `${table.toUpperCase()}-Partitionstabelle, 1 Volume:`
                : `${table.toUpperCase()}-Partitionstabelle, ${count} Volumes:`,
        contentsNoTable: (count: number) =>
            count === 1 ? 'Keine Partitionstabelle, 1 Volume:' : `Keine Partitionstabelle, ${count} Volumes:`,
        contentsRow: (label: string, fs: string, size: string) => `${label} — ${fs}, ${size}`,
        unnamedVolume: 'ohne Namen',
        notEmptyWarning: 'Diese Karte ist nicht leer. Das Schreiben zerstört alles oben Genannte.',
        confirmNotEmpty: (name: string, what: string) =>
            `Mir ist klar, dass ${name} gelöscht wird, einschließlich ${what}.`,
        andMore: (n: number) => `und ${n} weitere`,
        tooLarge: (needed: string, capacity: string) =>
            `Dieses Image schreibt ${needed}, die Karte fasst aber nur ${capacity}. Es passt nicht.`,
        sizeUnknown:
            'Wie viel dieses Image schreibt, steht erst beim Entpacken fest und wird daher während des Schreibens geprüft.',
        writes: (amount: string) => `schreibt ${amount}`,
    },

    assistant: {
        pickVolume: 'Einen Datenträger und ein lesbares Volume wählen, um den Assistenten zu nutzen.',
        heading: (label: string, fs: string) => `Assistent — ${label} (${fs})`,
        allowEdits: 'Änderungen erlauben',
        allowEditsHint: 'Öffnet den Datenträger neu, damit der Assistent Dateien ändern kann',
        newChat: 'Neues Gespräch',
        writableBanner: 'Der Assistent darf Dateien auf diesem Datenträger ändern. Es gibt kein Rückgängig.',
        backupIs: (path: string) => ` Sicherungskopie: ${path}`,
        noBackup: ' In dieser Sitzung wurde keine Sicherungskopie angelegt.',
        promptSuggestions: 'Eine Frage anklicken oder eine eigene tippen:',
        suggestions: [
            'Diese Karte lief nicht mehr. Finde heraus, was schiefgegangen ist.',
            'Was läuft hier drauf — welche Distribution, welcher Kernel, welche wichtigen Dienste?',
            'Lies das Ende der Logs in /var/log und fasse die letzten Fehler zusammen.',
            'Warum kam das Netzwerk nicht hoch? Sieh in den Logs und der Netzwerkkonfiguration nach.',
            'Zeig mir /etc/fstab und erkläre, was eingehängt wird.',
        ],
        placeholder: 'Frag etwas über die Dateien auf diesem Volume…',
        ask: 'Fragen',
        stop: 'Stoppen',
        working: 'arbeitet…',
        failed: 'fehlgeschlagen',
        apiKeyHeading: 'Claude-API-Schlüssel',
        apiKeyIntro:
            'Der Assistent spricht mit der Claude-API und braucht dafür einen Schlüssel. Setze ANTHROPIC_API_KEY in deiner Umgebung oder trage ihn hier ein — dann wird er in deiner Benutzerkonfiguration gespeichert, lesbar nur für dich.',
        apiKeySave: 'Speichern',
    },

    gate: {
        heading: 'Zuerst eine Sicherungskopie',
        intro: (path: string, size: string) =>
            `Du bist dabei, den Assistenten auf ${path} (${size}) schreiben zu lassen. Änderungen sind endgültig — es gibt kein Rückgängig.`,
        rationale: (isImage: boolean) =>
            `Der ext4-Schreibpfad in go-diskfs ist deutlich weniger erprobt als der Lesepfad, und in diesem Lesepfad haben wir vier Fehler gefunden. Lege deshalb zuerst eine vollständige Kopie ${isImage ? 'dieses Images' : 'dieser Karte'} an. Geht etwas schief, schreibst du die Kopie einfach zurück.`,
        haveBackup: (path: string, detail: string | null) =>
            `Sicherungskopie vorhanden: ${path}${detail ? ` — ${detail}` : ''}`,
        verified: (amount: string) => `${amount}, geprüft`,
        makeCopy: 'Jetzt eine Kopie anlegen…',
        makeAnother: 'Noch eine Kopie anlegen…',
        copying: 'Wird kopiert…',
        cancel: 'Abbrechen',
        acknowledgeWithBackup: 'Ich habe eine Sicherungskopie und will fortfahren.',
        acknowledgeWithout: 'Ich habe anderswo bereits eine Sicherungskopie, oder ich kann auf diesen Datenträger verzichten.',
        enable: 'Änderungen aktivieren',
    },

    fs: {
        unknown: 'unbekannt',
        noDriver: (name: string) => `Erkannt, aber es gibt keinen Lesetreiber für ${name}.`,
        noSignature:
            'Keine bekannte Dateisystem-Signatur gefunden — leer, beschädigt oder ein unbekanntes Format.',
        swap: 'Auslagerungsspeicher — hier liegen keine Dateien.',
        luks: 'Verschlüsselt. Diese Anwendung kann das nicht entsperren; nutze cryptsetup.',
        lvm: 'LVM-Container. Die eigentlichen Volumes liegen darin und sind so nicht lesbar.',
    },
}
