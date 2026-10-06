import {Messages} from './en'

/** The Dutch catalogue. Typed as Messages, so anything missing fails the build. */
export const nl: Messages = {
    localeName: 'Nederlands',

    app: {
        title: 'SD Card Sniffer',
        mode: {browse: 'Bladeren', copy: 'Kopiëren', assistant: 'Assistent'},
        notElevated: 'Niet als root gestart — ruwe schijftoegang zal mislukken. Start met sudo.',
        pickDisk: 'Kies links een schijf om de volumes te bekijken.',
    },

    devices: {
        heading: 'Apparaten',
        refresh: 'Vernieuwen',
        none: 'Geen schijven gevonden.',
        removable: 'verwisselbaar',
        internal: 'intern',
        unmount: (id: string) => `Ontkoppelen (${id})`,
        unmounted: (id: string) => `${id} is ontkoppeld.`,
        openImage: 'Disk-image openen…',
    },

    volume: {
        unnamed: (n: number) => `Volume ${n}`,
    },

    browser: {
        up: 'Eén niveau omhoog',
        exportDir: 'Map exporteren',
        exporting: 'Exporteren…',
        loading: 'Laden…',
        emptyDir: 'Lege map.',
        colName: 'Naam',
        colSize: 'Grootte',
        colMode: 'Rechten',
        colModified: 'Gewijzigd',
        exported: (files: number, dest: string, skipped: number) =>
            skipped > 0
                ? `${files} bestanden naar ${dest}, ${skipped} overgeslagen`
                : `${files} bestanden naar ${dest}`,
        savedAs: (dest: string) => `Opgeslagen als ${dest}`,
    },

    preview: {
        pickFile: 'Selecteer een bestand om de inhoud te bekijken.',
        modeAuto: 'auto',
        modeText: 'tekst',
        modeHex: 'hex',
        save: 'Opslaan als…',
        previous: 'Vorige',
        next: 'Volgende',
        range: (from: string, to: string, total: string) => `byte ${from}–${to} van ${total}`,
    },

    copy: {
        pickMedium: 'Kies links een kaart, of open een image, om te kopiëren of te schrijven.',
        opened: 'Geopend:',
        cloneHeadingCard: 'Kaart kopiëren naar image',
        cloneHeadingImage: 'Image kopiëren naar nieuw image',
        cloneIntroCard: 'Leest de kaart uit naar een bestand. De kaart zelf blijft ongewijzigd.',
        cloneIntroImage: 'Leest dit image uit naar een bestand. Het origineel blijft ongewijzigd.',
        trim: 'Stoppen na de laatste partitie',
        trimDetail: (end: string, total: string, saved: string | null) =>
            saved ? ` — ${end} in plaats van ${total}, scheelt ${saved}` : ` — ${end} in plaats van ${total}`,
        compress: 'Comprimeren (gzip)',
        verify: 'Achteraf verifiëren',
        verifyDetailImage: ' — leest het image terug en vergelijkt de hash',
        verifyDetailCard: ' — leest de kaart terug en vergelijkt de hash',
        gptWarning:
            'Dit medium gebruikt GPT. Een afgekapt image mist de reserve-partitietabel aan het eind van de schijf; gdisk kan die na het terugschrijven herstellen.',
        startCard: 'Kaart kopiëren…',
        startImage: 'Image kopiëren…',
        cancel: 'Annuleren',
        phaseCopying: 'Kopiëren',
        phaseWriting: 'Schrijven',
        phaseVerifying: 'Verifiëren',
        remaining: (t: string) => `nog ${t}`,
        result: (amount: string, dest: string, onDisk: string | null, elapsed: string, verified: boolean) =>
            `${amount} naar ${dest}${onDisk ? ` (${onDisk} op schijf)` : ''} in ${elapsed}.${verified ? ' Geverifieerd.' : ''}`,
    },

    write: {
        heading: 'Image naar kaart schrijven',
        intro: 'Overschrijft de hele kaart. Alles wat erop staat is daarna weg.',
        notRemovable: (name: string) =>
            `${name} is geen verwisselbare schijf. Schrijven is hierop geblokkeerd — ook als je het via de API zou proberen.`,
        needTarget:
            'Kies links een kaart als doel. Schrijven vraagt om een verwisselbaar apparaat; een geopend image kan alleen de bron zijn.',
        pick: 'Image kiezen…',
        pickAnother: 'Ander image kiezen…',
        confirm: (name: string, node: string, size: string) =>
            `Ik weet dat ${name} (${node}, ${size}) volledig gewist wordt.`,
        start: 'Schrijven naar kaart',

        contentsHeading: 'Staat nu op deze kaart',
        contentsEmpty: 'Deze kaart heeft geen partitietabel en geen herkenbaar filesystem. Hij lijkt leeg.',
        contentsUnreadable: (reason: string) =>
            `De inhoud van deze kaart kon niet gelezen worden (${reason}). Wat erop staat wordt hoe dan ook gewist.`,
        contentsChecking: 'Kaart uitlezen…',
        contentsTable: (table: string, count: number) =>
            count === 1
                ? `${table.toUpperCase()}-partitietabel, 1 volume:`
                : `${table.toUpperCase()}-partitietabel, ${count} volumes:`,
        contentsNoTable: (count: number) =>
            count === 1 ? 'Geen partitietabel, 1 volume:' : `Geen partitietabel, ${count} volumes:`,
        contentsRow: (label: string, fs: string, size: string) => `${label} — ${fs}, ${size}`,
        unnamedVolume: 'naamloos',
        notEmptyWarning: 'Deze kaart is niet leeg. Schrijven vernietigt alles wat hierboven staat.',
        confirmNotEmpty: (name: string, what: string) =>
            `Ik weet dat ${name} gewist wordt, inclusief ${what}.`,
        andMore: (n: number) => `en nog ${n}`,
        tooLarge: (needed: string, capacity: string) =>
            `Dit image schrijft ${needed}, maar de kaart biedt maar ${capacity}. Het past niet.`,
        sizeUnknown:
            'Hoeveel dit image schrijft is pas tijdens het uitpakken bekend, dus dat wordt onder het schrijven gecontroleerd.',
        writes: (amount: string) => `schrijft ${amount}`,
        format: {
            raw: 'onbewerkt',
            gzip: 'gzip',
            xz: 'xz',
            bzip2: 'bzip2',
            zip: 'zip',
        },
    },

    assistant: {
        pickVolume: 'Kies een schijf en een leesbaar volume om de assistent te gebruiken.',
        heading: (label: string, fs: string) => `Assistent — ${label} (${fs})`,
        allowEdits: 'Wijzigen toestaan',
        allowEditsHint: 'Heropent de schijf zodat de assistent bestanden kan wijzigen',
        newChat: 'Nieuw gesprek',
        writableBanner: 'De assistent mag bestanden op dit medium veranderen. Er is geen ongedaan maken.',
        backupIs: (path: string) => ` Reservekopie: ${path}`,
        noBackup: ' Er is deze sessie geen reservekopie gemaakt.',
        promptSuggestions: 'Klik een vraag om te beginnen, of typ je eigen:',
        suggestions: [
            'Deze kaart deed het niet meer. Zoek uit wat er misging.',
            'Wat draait hierop — welke distributie, kernel en belangrijkste diensten?',
            'Bekijk het einde van de logs in /var/log en vat de laatste fouten samen.',
            'Waarom kwam het netwerk niet op? Kijk in de logs en de netwerkconfiguratie.',
            'Laat /etc/fstab zien en leg uit wat er gemount wordt.',
        ],
        placeholder: 'Vraag iets over de bestanden op dit volume…',
        ask: 'Vragen',
        stop: 'Stoppen',
        working: 'bezig…',
        failed: 'mislukt',
        apiKeyHeading: 'Claude API-sleutel',
        apiKeyIntro:
            'De assistent praat met de Claude API en heeft daarvoor een sleutel nodig. Zet ANTHROPIC_API_KEY in je omgeving, of vul hem hier in — dan wordt hij opgeslagen in je gebruikersconfiguratie, leesbaar voor jou alleen.',
        apiKeySave: 'Opslaan',
    },

    gate: {
        heading: 'Eerst een reservekopie',
        intro: (path: string, size: string) =>
            `Je staat op het punt de assistent te laten schrijven naar ${path} (${size}). Wijzigingen zijn definitief — er is geen ongedaan maken.`,
        rationale: (isImage: boolean) =>
            `De ext4-schrijfroutine van go-diskfs is aanzienlijk minder beproefd dan de leesroutine, en in die leesroutine vonden we vier bugs. Maak daarom eerst een volledige kopie van ${isImage ? 'dit image' : 'deze kaart'}. Gaat er iets mis, dan schrijf je die kopie gewoon terug.`,
        haveBackup: (path: string, detail: string | null) =>
            `Reservekopie aanwezig: ${path}${detail ? ` — ${detail}` : ''}`,
        verified: (amount: string) => `${amount}, geverifieerd`,
        makeCopy: 'Nu een kopie maken…',
        makeAnother: 'Nog een kopie maken…',
        copying: 'Kopiëren…',
        cancel: 'Annuleren',
        acknowledgeWithBackup: 'Ik heb een reservekopie en wil verder.',
        acknowledgeWithout: 'Ik heb elders al een reservekopie, of ik kan dit medium missen.',
        enable: 'Wijzigen inschakelen',
    },

    fs: {
        unknown: 'onbekend',
        noDriver: (name: string) => `Herkend, maar er is geen leesdriver voor ${name}.`,
        noSignature:
            'Geen bekende filesystem-signature gevonden — leeg, beschadigd of een onbekend formaat.',
        swap: 'Wisselgeheugen — hier staan geen bestanden op.',
        luks: 'Versleuteld. Ontgrendelen kan deze app niet; gebruik cryptsetup.',
        lvm: 'LVM-container. De echte volumes zitten hierbinnen en zijn zo niet leesbaar.',
    },
}
