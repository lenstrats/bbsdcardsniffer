package i18n

import "fmt"

var dutch = Messages{
	NoDeviceOpen: func() string { return "er is geen apparaat geopend" },
	NoPermission: func(path string) string {
		return fmt.Sprintf("geen toestemming om %s te lezen: start deze applicatie als root of Administrator", path)
	},
	DeviceBusy: func(path string) string {
		return fmt.Sprintf("%s is in gebruik: ontkoppel eerst de volumes ervan", path)
	},
	NoSuchVolume: func(index int) string {
		return fmt.Sprintf("volume %d bestaat niet op deze schijf", index+1)
	},
	VolumeUnreadable: func(index int, kind string) string {
		return fmt.Sprintf("volume %d bevat %s, waar niet in te bladeren is", index+1, kind)
	},
	FSUnsupported: func(kind string) string {
		return fmt.Sprintf("%s kan deze applicatie niet lezen", kind)
	},
	DiskReadOnly: func() string {
		return "deze schijf is alleen-lezen geopend; schakel eerst wijzigen in"
	},
	CannotShorten: func(path string, size int) string {
		return fmt.Sprintf("%s: dit filesystem kan een bestand niet inkorten, waardoor de oude inhoud voorbij byte %d zou blijven staan", path, size)
	},

	UnmountUnsupported: func() string {
		return "ontkoppelen vanuit deze applicatie werkt alleen op macOS"
	},
	UnmountFailed: func(id, detail string) string {
		return fmt.Sprintf("kon %s niet ontkoppelen: %s", id, detail)
	},

	TransferRunning: func(kind string) string {
		return fmt.Sprintf("er loopt al een bewerking (%s)", kind)
	},
	NoTransferRunning: func() string { return "er loopt geen bewerking" },
	CloneNeedsBoth: func() string {
		return "kopiëren vraagt zowel een bron als een doelbestand"
	},
	WriteNeedsBoth: func() string { return "schrijven vraagt zowel een image als een kaart" },
	NoTrimTable: func(detail string) string {
		return fmt.Sprintf("kan de partitietabel niet lezen om de kopie af te kappen: %s", detail)
	},
	BackupOverwrites: func() string {
		return "de reservekopie kan niet over het image heen dat hij kopieert"
	},
	NotRemovableFixed: func(node, name string) string {
		return fmt.Sprintf("%s (%s) is een vaste schijf: schrijven naar een niet-verwisselbare schijf wordt geweigerd", node, name)
	},
	NotRemovableSystem: func(node, name string) string {
		return fmt.Sprintf("%s (%s) is een interne schijf: schrijven naar een niet-verwisselbare schijf wordt geweigerd", node, name)
	},
	NotRemovableUnknown: func(path string) string {
		return fmt.Sprintf("%s staat niet in de lijst met aangesloten schijven: schrijven wordt geweigerd", path)
	},
	ImageTooLarge: func(needed, capacity int64) string {
		return fmt.Sprintf("het image is groter dan de kaart: er is meer dan %d bytes nodig maar de kaart biedt %d", needed, capacity)
	},
	VerifyFailedClone: func(want, got string) string {
		return fmt.Sprintf("verificatie mislukt: de kaart gaf hash %s maar het image leest terug als %s", want, got)
	},
	VerifyFailedWrite: func(want, got string) string {
		return fmt.Sprintf("verificatie mislukt: %s is geschreven maar de kaart leest terug als %s", want, got)
	},

	NoAPIKey:      func() string { return "er is geen Claude API-sleutel ingesteld" },
	AssistantBusy: func() string { return "de assistent is nog met de vorige vraag bezig" },
	AssistantIdle: func() string { return "de assistent is nergens mee bezig" },
	NoSuchTool:    func() string { return "die tool bestaat niet" },
	OutOfToolTurns: func(rounds int) string {
		return fmt.Sprintf("na %d rondes tool-aanroepen opgegeven zonder tot een antwoord te komen", rounds)
	},
}
