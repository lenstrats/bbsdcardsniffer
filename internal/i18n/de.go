package i18n

import "fmt"

var german = Messages{
	NoDeviceOpen: func() string { return "es ist kein Gerät geöffnet" },
	NoPermission: func(path string) string {
		return fmt.Sprintf("keine Berechtigung, %s zu lesen: starte diese Anwendung als root oder Administrator", path)
	},
	DeviceBusy: func(path string) string {
		return fmt.Sprintf("%s ist in Benutzung: hänge zuerst seine Volumes aus", path)
	},
	NoSuchVolume: func(index int) string {
		return fmt.Sprintf("Volume %d gibt es auf diesem Datenträger nicht", index+1)
	},
	VolumeUnreadable: func(index int, kind string) string {
		return fmt.Sprintf("Volume %d enthält %s, worin sich nicht blättern lässt", index+1, kind)
	},
	FSUnsupported: func(kind string) string {
		return fmt.Sprintf("%s kann diese Anwendung nicht lesen", kind)
	},
	DiskReadOnly: func() string {
		return "dieser Datenträger ist schreibgeschützt geöffnet; aktiviere zuerst Änderungen"
	},
	CannotShorten: func(path string, size int) string {
		return fmt.Sprintf("%s: dieses Dateisystem kann eine Datei nicht kürzen, wodurch der alte Inhalt hinter Byte %d stehen bliebe", path, size)
	},

	UnmountUnsupported: func() string {
		return "das Aushängen aus dieser Anwendung heraus geht nur unter macOS"
	},
	UnmountFailed: func(id, detail string) string {
		return fmt.Sprintf("%s konnte nicht ausgehängt werden: %s", id, detail)
	},

	TransferRunning: func(kind string) string {
		return fmt.Sprintf("es läuft bereits ein Vorgang (%s)", kind)
	},
	NoTransferRunning: func() string { return "es läuft kein Vorgang" },
	CloneNeedsBoth: func() string {
		return "zum Kopieren werden eine Quelle und eine Zieldatei gebraucht"
	},
	WriteNeedsBoth: func() string { return "zum Schreiben werden ein Image und eine Karte gebraucht" },
	NoTrimTable: func(detail string) string {
		return fmt.Sprintf("die Partitionstabelle lässt sich nicht lesen, um die Kopie abzuschneiden: %s", detail)
	},
	BackupOverwrites: func() string {
		return "die Sicherungskopie kann nicht das Image überschreiben, das sie kopiert"
	},
	NotRemovableFixed: func(node, name string) string {
		return fmt.Sprintf("%s (%s) ist eine feste Platte: das Schreiben auf einen nicht wechselbaren Datenträger wird verweigert", node, name)
	},
	NotRemovableSystem: func(node, name string) string {
		return fmt.Sprintf("%s (%s) ist eine interne Platte: das Schreiben auf einen nicht wechselbaren Datenträger wird verweigert", node, name)
	},
	NotRemovableUnknown: func(path string) string {
		return fmt.Sprintf("%s steht nicht in der Liste der angeschlossenen Datenträger: das Schreiben wird verweigert", path)
	},
	ImageTooLarge: func(needed, capacity int64) string {
		return fmt.Sprintf("das Image ist größer als die Karte: es braucht mehr als %d Bytes, die Karte fasst aber %d", needed, capacity)
	},
	VerifyFailedClone: func(want, got string) string {
		return fmt.Sprintf("Prüfung fehlgeschlagen: die Karte ergab %s, das Image liest sich aber als %s zurück", want, got)
	},
	VerifyFailedWrite: func(want, got string) string {
		return fmt.Sprintf("Prüfung fehlgeschlagen: %s wurde geschrieben, die Karte liest sich aber als %s zurück", want, got)
	},

	NoAPIKey:      func() string { return "es ist kein Claude-API-Schlüssel eingerichtet" },
	AssistantBusy: func() string { return "der Assistent arbeitet noch an der vorigen Frage" },
	AssistantIdle: func() string { return "der Assistent arbeitet gerade an nichts" },
	NoSuchTool:    func() string { return "dieses Werkzeug gibt es nicht" },
	OutOfToolTurns: func(rounds int) string {
		return fmt.Sprintf("nach %d Runden von Werkzeugaufrufen aufgegeben, ohne zu einer Antwort zu kommen", rounds)
	},
}
