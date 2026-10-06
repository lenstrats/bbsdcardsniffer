package main

import (
	"strings"
	"testing"

	"bbsdcardsniffer/internal/i18n"
)

// The backend phrases its own errors, so switching the language has to change
// what reaches the interface — not only the labels the frontend draws.
func TestBackendErrorsFollowTheLocale(t *testing.T) {
	t.Cleanup(func() { i18n.Set(i18n.English) })

	app := NewApp("")
	defer app.shutdown(nil)

	app.SetLocale("en")
	_, enErr := app.ListDir(0, "/")
	if enErr == nil {
		t.Fatal("listing with nothing open should fail")
	}

	app.SetLocale("nl")
	_, nlErr := app.ListDir(0, "/")
	if nlErr == nil {
		t.Fatal("listing with nothing open should fail")
	}

	if enErr.Error() == nlErr.Error() {
		t.Fatalf("the message did not change with the locale: %q", enErr)
	}
	if !strings.Contains(enErr.Error(), "no device is open") {
		t.Errorf("English message = %q", enErr)
	}
	if !strings.Contains(nlErr.Error(), "geen apparaat geopend") {
		t.Errorf("Dutch message = %q", nlErr)
	}

	app.SetLocale("de")
	_, deErr := app.ListDir(0, "/")
	if deErr == nil {
		t.Fatal("listing with nothing open should fail")
	}
	if !strings.Contains(deErr.Error(), "kein Gerät geöffnet") {
		t.Errorf("German message = %q", deErr)
	}
	if deErr.Error() == nlErr.Error() || deErr.Error() == enErr.Error() {
		t.Error("the German message is not distinct from the others")
	}
}

// An unknown tag must leave the application in a working language rather than
// with empty messages.
func TestUnknownLocaleFallsBack(t *testing.T) {
	t.Cleanup(func() { i18n.Set(i18n.English) })

	app := NewApp("")
	defer app.shutdown(nil)

	app.SetLocale("nl")
	app.SetLocale("does-not-exist")

	_, err := app.ListDir(0, "/")
	if err == nil {
		t.Fatal("listing with nothing open should fail")
	}
	if !strings.Contains(err.Error(), "no device is open") {
		t.Errorf("did not fall back to English: %q", err)
	}
}

// The refusal to write to a fixed disk is one of the messages a user is most
// likely to meet, so it is checked in both languages against a real disk.
func TestWriteRefusalIsTranslated(t *testing.T) {
	t.Cleanup(func() { i18n.Set(i18n.English) })
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)

	app.SetLocale("nl")
	_, err := app.WriteImage(WriteRequest{Image: img, Device: "/dev/not-a-disk"})
	if err == nil {
		t.Fatal("writing to a nonexistent target should fail")
	}
	if !strings.Contains(err.Error(), "geweigerd") {
		t.Errorf("Dutch refusal = %q", err)
	}
}
