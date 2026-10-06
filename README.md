# SD Card Sniffer

Read-only inspectie van SD-kaarten en disk-images, inclusief Linux-filesystems
die macOS en Windows zelf niet kunnen mounten. Go-backend (Wails v2) met een
React-frontend.

De applicatie opent de kaart altijd read-only en schrijft er nooit naartoe.

## Wat het kan lezen

| Filesystem | Bladeren | Opmerking |
|---|---|---|
| ext2 / ext3 / ext4 | ja | de root-partitie van een Raspberry Pi-kaart |
| FAT12 / FAT16 / FAT32 | ja | de boot-partitie |
| ISO 9660 | ja | |
| SquashFS | ja | |
| btrfs, XFS, F2FS, NTFS, exFAT, APFS, HFS+ | nee | wordt wel herkend en benoemd |
| LUKS1 / LUKS2 | nee | versleuteld; ontgrendelen via `cryptsetup` |
| LVM2 | nee | container; de volumes erbinnen zijn zo niet bereikbaar |
| Linux swap | n.v.t. | wordt herkend, bevat geen bestanden |

Partitietabellen: GPT en MBR. Een kaart zonder partitietabel wordt als één
volume behandeld.

## Gebruiken

Ruwe schijftoegang vereist verhoogde rechten:

```sh
sudo ./build/bin/bbsdcardsniffer.app/Contents/MacOS/bbsdcardsniffer
```

Optioneel meteen een apparaat of image openen:

```sh
sudo ./build/bin/bbsdcardsniffer.app/Contents/MacOS/bbsdcardsniffer /dev/disk4
./build/bin/bbsdcardsniffer.app/Contents/MacOS/bbsdcardsniffer ~/dumps/card.img
```

Een image-bestand (bijvoorbeeld een `dd`-dump) heeft geen root nodig en is ook
via de knop **Disk-image openen…** te kiezen.

Op macOS moet een gemounte kaart eerst worden ontkoppeld. De knop
**Ontkoppelen** naast het apparaat doet dat via `diskutil unmountDisk`; de
kaart blijft daarna gewoon in de reader zitten.

## Klonen en schrijven

Naast bladeren heeft de app een **Kopiëren**-modus met twee richtingen.

**Kopiëren naar een image.** Leest uit naar een bestand; het origineel blijft
ongewijzigd. De bron is het medium dat open staat — een kaart óf een geopend
image, zodat je ook een kopie van een kopie kunt maken. Opties:

- *Stoppen na de laatste partitie* — scheelt het lege staartstuk van een kaart
  waarvan de rootfs nog niet is uitgerekt. Bij GPT waarschuwt de app dat de
  reserve-partitietabel dan buiten het image valt (`gdisk` herstelt die).
- *Comprimeren* naar gzip.
- *Verifiëren* — leest het image terug en vergelijkt de SHA-256 met wat er van
  de kaart kwam.

**Image → kaart (schrijven).** Het doel is altijd de kaart die links
geselecteerd staat; een geopend image kan alleen de bron zijn. Overschrijft de
kaart volledig.

Voor het schrijven wordt de doelkaart uitgelezen en getoond wat erop staat:
partitietabel, volumes, filesystem en label. Is de kaart niet leeg, dan noemt de
bevestiging bij naam wat er verdwijnt — *"Ik weet dat disk4 gewist wordt,
inclusief boot, rootfs"* — in plaats van naar een anoniem apparaat te vragen.
Een kaart die niet uitgelezen kan worden wordt niet als leeg gepresenteerd maar
als onleesbaar, met de waarschuwing dat er hoe dan ook gewist wordt.

Past het image niet op de kaart, dan wordt de knop geblokkeerd voordat er iets
gebeurt — voor zover de grootte vooraf te kennen is. Een onbewerkt image en een
zip weten dat; xz en gzip niet (gzip bewaart de lengte modulo 4 GiB, precies in
het bereik waar het antwoord ertoe doet), dus daar valt de controle terug op het
schrijfmoment zelf. Ondersteunt
onbewerkte images en `.gz`, `.xz`, `.bz2` en `.zip`; het formaat wordt aan de
inhoud herkend, niet aan de extensie, zodat een hernoemd bestand niet rauw naar
de kaart gaat.

Schrijven is met opzet streng afgeschermd:

- `internal/imaging` weigert elk doel dat niet als verwisselbaar in de
  schijflijst staat, en weigert ook een doel dat er helemaal niet in staat.
  Dat is een harde weigering in de backend, niet een dialoog in de UI.
- De kaart wordt eerst ontkoppeld (macOS `diskutil unmountDisk`, Linux
  `umount`, Windows `FSCTL_LOCK_VOLUME` + `FSCTL_DISMOUNT_VOLUME`).
- De UI vraagt om een expliciete bevestiging waarin het doelapparaat bij naam
  wordt genoemd; die bevestiging vervalt zodra je een andere kaart kiest.
- Een ongeldig doel wordt geweigerd vóórdat er iets wordt afgebroken, zodat een
  vergissing je geopende schijf niet kost.
- Na afloop wordt de kaart op macOS uitgeworpen, zodat de flush zeker klaar is
  voordat hij eruit kan.

Beide richtingen zijn te annuleren en tonen doorvoer en resterende tijd.

> Het schrijfpad op **Windows** is geschreven volgens de gedocumenteerde
> lock/dismount-volgorde maar is niet op Windows getest — er was geen Windows
> machine beschikbaar. macOS en Linux volgen hetzelfde patroon met `diskutil`
> respectievelijk `umount`.

## Assistent

Een derde modus zet een Claude-agent op het geopende volume. Je stelt een vraag
in gewone taal — "zoek in /var/log waarom het netwerk niet opkwam" — en het
model roept zelf de tools aan die het nodig heeft. Dat patroon heet *tool use*;
de lus eromheen een *agentic loop*.

De tool surface:

| Tool | Doet | Wijzigt |
|---|---|---|
| `list_dir` | mapinhoud tonen | nee |
| `read_file` | bestand lezen, gepagineerd, binair wordt gemeld niet gedumpt | nee |
| `find` | bestandsnamen zoeken in de boom | nee |
| `grep` | regels zoeken in tekstbestanden, met bestand en regelnummer | nee |
| `stat` | grootte, rechten, wijzigingsdatum | nee |
| `write_file` | volledige inhoud van een bestand vervangen | **ja** |
| `make_dir` | map aanmaken | **ja** |
| `delete` | bestand of lege map verwijderen | **ja** |

Voor het diagnosewerk waar dit vooral voor bedoeld is, zijn de leestools daarop
toegesneden:

- `read_file` neemt een `tail`-argument. Logs groeien aan het eind, dus de
  storing staat achteraan; vooruit paginaeren door een syslog van 200 MB is
  geen werkbare manier om te zoeken.
- Geroteerde logs (`dmesg.4.gz`) worden automatisch uitgepakt. Zonder dat zien
  ze eruit als binair en verdwijnt stilzwijgend alle geschiedenis van vóór het
  huidige bestand.
- `grep` slaat grote bestanden niet over maar doorzoekt hun staart, en markeert
  de regelnummers dan als bij benadering.
- De systemd-journal (`/var/log/journal`) is een binair formaat dat deze tool
  niet kan decoderen. In plaats van "binair bestand" krijgt het model te horen
  wát het is en waar de tekstlogs staan, zodat het antwoord dat kan doorgeven.

Standaard is de schijf alleen-lezen en worden de drie schrijftools **niet eens
aan het model getoond** — het kan er dus niet naar grijpen. Pas als je
*Wijzigen toestaan* aanzet wordt het medium heropend in schrijfmodus en komen ze
beschikbaar. Elke aanroep verschijnt in het gesprek, wijzigende aanroepen rood
gemarkeerd, met de volledige uitkomst uitklapbaar.

Een sleutel is nodig: `ANTHROPIC_API_KEY` in je omgeving, of eenmalig invullen
in de app, waarna hij in je gebruikersconfiguratie staat met rechten `0600`.

### Reservekopie vóór wijzigen

*Wijzigen toestaan* schakelt niet meteen om, maar opent eerst een poort die
aanbiedt een volledige kopie van het medium te maken — kaart of image, één knop,
met verificatie achteraf. Doorgaan zonder kopie kan wel, maar vereist een
expliciet vinkje; het is jouw medium. Is er eenmaal een kopie, dan onthoudt de
app waar die staat en noemt hem in de waarschuwingsbalk zolang schrijven aan
staat.

Die stap staat er niet voor de vorm. De ext4-schrijfroutine van go-diskfs is
aanzienlijk minder beproefd dan de leesroutine, en in die leesroutine vonden we
vier bugs. Gaat er iets mis, dan schrijf je de kopie gewoon terug — dat pad is
getest.

## Talen (i18n)

De interface is Engels, Nederlands en Duits; de taal is te kiezen rechtsboven en
wordt onthouden. **Engels is de standaard** — ook op een Nederlandstalig
systeem. De terminologie waar deze app mee werkt (partitietabel, superblock,
filesystem) is in elke naslagbron Engels, en een andere taal hoort een bewuste
keuze te zijn in plaats van een gevolg van de machine waarop hij toevallig
draait.

De catalogi staan in [frontend/src/i18n/](frontend/src/i18n/):

| Bestand | Rol |
|---|---|
| `en.ts` | de Engelse teksten én de vorm waar elke taal aan moet voldoen |
| `nl.ts`, `de.ts` | Nederlands en Duits, beide getypeerd als `Messages` |
| `index.ts` | context, `useT()`-hook, taalkeuze en opslag |

Een taal toevoegen is één bestand naast `nl.ts` plus een regel in `locales`.
Omdat elke catalogus als `Messages` getypeerd is, is een ontbrekende of
verkeerd gespelde sleutel een **buildfout**, geen gat in de UI. Hetzelfde geldt
voor de lijst met voorbeeldvragen: die is een tuple van vaste lengte, dus een
vraag toevoegen in één taal en vergeten in een andere loopt vast op `tsc`.

Teksten met variabelen zijn functies in plaats van sjablonen met plaatshouders,
zodat de woordvolgorde aan de vertaler is en niet aan de Engelse zinsbouw.

De backend heeft een eigen catalogus in [internal/i18n/](internal/i18n/), want
foutmeldingen worden daar geformuleerd. De frontend geeft de taalkeuze door met
`SetLocale`, zodat beide helften dezelfde taal spreken. De twee catalogi
overlappen niet: de frontend doet de interface, Go de meldingen.

Go kan een ontbrekend veld niet als buildfout aanmerken zoals TypeScript, dus
dat gat wordt gedicht door een test die met reflectie over `Messages` loopt en
controleert dat elke taal elk veld heeft, dat geen enkele melding leeg is, en
dat geen enkele Nederlandse tekst identiek is aan de Engelse — dat laatste is
vrijwel altijd een vergeten vertaling.

Ook aan Go-zijde **niet** vertaald: meldingen die uit de filesystem-drivers
komen (`invalid checksum type 0`, `resource busy`). Die zijn diagnostisch in
plaats van bruikbaar, ze zijn zo op te zoeken, en de tekst van een externe
library herschrijven maakt dat alleen moeilijker. Ze reizen mee als detail
naast een vertaalde uitleg:

```
geen toestemming om /dev/rdisk4 te lezen: start deze applicatie als root
of Administrator (open /dev/rdisk4: permission denied)
```

## Gepatchte go-diskfs

`vendor/` bevat een gepatchte go-diskfs v1.9.4. Drie checksum-controles in de
ext4-reader gingen ervan uit dat het `metadata_csum`-feature aanstaat, terwijl
elke kaart die met een oudere `mkfs.ext4` is gemaakt dat feature niet heeft — en
dat is het merendeel van de kaarten in het veld. Zonder de patches faalt zo'n
volume met `invalid checksum type 0`.

De patch staat in [patches/](patches/) en raakt drie plekken:

| Bestand | Probleem |
|---|---|
| `superblock.go` | `s_checksum_type` werd onvoorwaardelijk afgekeurd als hij niet 1 is, terwijl het veld zonder `metadata_csum` betekenisloos is en verderop nooit meer wordt gelezen |
| `groupdescriptors.go` | de CRC16-variant voor `gdt_csum` is verkeerd geïmplementeerd (verkeerde CRC-16-variant, verkeerde seed, en de checksum wordt over zijn eigen veld berekend), dus elke descriptor mismatcht |
| `inode.go` | inode-checksums werden altijd geverifieerd; elke naburige controle in dat bestand is wél op `metadata_csum` afgeschermd, deze niet |
| `directoryentry.go` | `Info()` gaf alleen de type-bits terug, waardoor elk bestand als `----------` verscheen; de rechten worden er wel degelijk uit de inode geparsed |

Filesystems die `metadata_csum` wél gebruiken worden onverminderd volledig
geverifieerd, dus echte corruptie wordt nog steeds gevonden.

Na het wijzigen van dependencies niet `go mod vendor` draaien maar:

```sh
./scripts/vendor.sh
```

Dat vendort opnieuw en zet de patches er weer overheen.

## Ontwikkelen

```sh
wails dev      # live reload
wails build    # productiebuild in build/bin
go test ./...  # backend-tests
```

De tests bouwen zelf een GPT-image met een FAT32-boot- en een ext4-rootpartitie,
dus er is geen kaart nodig om ze te draaien. Een sample-image voor de GUI:

```sh
SAMPLE_IMAGE=/tmp/card.img go test -run TestWriteSampleImage ./internal/volume/
```

Een echte kaart of image erdoorheen halen, met een verslag van wat er gevonden
en gelezen kon worden:

```sh
REAL_IMAGE=~/card.img go test -v -run TestRealImage ./internal/volume/
```

## Opbouw

| Pad | Rol |
|---|---|
| [app.go](app.go) | de methodes die aan de frontend gebonden zijn |
| [internal/blockdev/](internal/blockdev/) | ruwe device-toegang met sector-aligned reads en een blokcache |
| [internal/device/](internal/device/) | schijfoverzicht per platform (`diskutil`, `lsblk`, `Get-Disk`) |
| [internal/volume/](internal/volume/) | partitietabel, filesystem-herkenning, bladeren en exporteren |
| [internal/imaging/](internal/imaging/) | klonen, schrijven, decompressie en voortgang — het enige dat naar een apparaat schrijft |
| [internal/assistant/](internal/assistant/) | de Claude-agent: tool surface en de agentic loop |
| [internal/config/](internal/config/) | opslag van de API-sleutel |
| [assistant.go](assistant.go) | de assistent-methodes die aan de frontend gebonden zijn |
| [transfer.go](transfer.go) | de kloon- en schrijfmethodes die aan de frontend gebonden zijn |
| [frontend/src/](frontend/src/) | React-UI: apparatenlijst, volumetabs, bestandsbrowser, hex/tekst-preview |

`internal/blockdev` opent standaard alleen-lezen en zijn `Writable()` weigert
dan; schrijven vereist expliciet `OpenReadWrite`. Rauwe apparaten accepteren
alleen hele sectoren, dus een gedeeltelijke schrijfactie wordt een
read-modify-write van de omliggende sectoren, met invalidatie van de blokcache.

De aligned-read-laag in `internal/blockdev` is niet optioneel: `/dev/rdiskN` op
macOS en `\\.\PhysicalDriveN` op Windows weigeren reads die niet op een sector
uitgelijnd zijn, terwijl de filesystem-drivers op willekeurige offsets lezen.
