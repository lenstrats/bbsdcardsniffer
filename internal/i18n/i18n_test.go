package i18n

import (
	"reflect"
	"testing"
)

// TestEveryLocaleIsComplete is what makes the struct-of-functions layout safe:
// Go cannot fail the build on a missing field the way TypeScript can, so the
// check happens here instead. A field left unset would be a nil func and would
// panic the moment that error occurred — which is exactly when a panic is
// least welcome.
func TestEveryLocaleIsComplete(t *testing.T) {
	shape := reflect.TypeOf(Messages{})

	for tag, catalogue := range catalogues {
		v := reflect.ValueOf(catalogue)
		for i := 0; i < shape.NumField(); i++ {
			if v.Field(i).IsNil() {
				t.Errorf("locale %q has no %s", tag, shape.Field(i).Name)
			}
		}
	}
}

// Every message must actually produce something, and a translation that came
// out identical to the English is almost always a forgotten one. Every locale
// is checked against English, so adding a language extends this automatically.
func TestMessagesProduceDistinctText(t *testing.T) {
	shape := reflect.TypeOf(Messages{})
	enValue := reflect.ValueOf(english)

	for tag, catalogue := range catalogues {
		if tag == English {
			continue
		}
		value := reflect.ValueOf(catalogue)

		for i := 0; i < shape.NumField(); i++ {
			field := shape.Field(i)
			if enValue.Field(i).IsNil() || value.Field(i).IsNil() {
				continue // already reported by the completeness test
			}

			args := sampleArgs(field.Type)
			enText := enValue.Field(i).Call(args)[0].String()
			text := value.Field(i).Call(args)[0].String()

			if enText == "" {
				t.Errorf("%s produces an empty string in English", field.Name)
			}
			if text == "" {
				t.Errorf("%s produces an empty string in %q", field.Name, tag)
			}
			if enText == text {
				t.Errorf("%s is identical in %q and English, which suggests it was not translated: %q", field.Name, tag, enText)
			}
		}
	}
}

// sampleArgs builds plausible arguments for a message function by its type, so
// the tests above can call every message without knowing its signature.
func sampleArgs(fn reflect.Type) []reflect.Value {
	args := make([]reflect.Value, fn.NumIn())
	for i := 0; i < fn.NumIn(); i++ {
		switch fn.In(i).Kind() {
		case reflect.String:
			args[i] = reflect.ValueOf("sample")
		case reflect.Int:
			args[i] = reflect.ValueOf(1)
		case reflect.Int64:
			args[i] = reflect.ValueOf(int64(1))
		default:
			args[i] = reflect.Zero(fn.In(i))
		}
	}
	return args
}

func TestSetFallsBackToEnglish(t *testing.T) {
	t.Cleanup(func() { Set(English) })

	for tag, catalogue := range catalogues {
		Set(tag)
		if T().NoDeviceOpen() != catalogue.NoDeviceOpen() {
			t.Errorf("Set(%q) did not switch the catalogue", tag)
		}
	}

	// An unknown tag must not leave the application without messages.
	Set(Locale("kl"))
	if T().NoDeviceOpen() != english.NoDeviceOpen() {
		t.Error("an unknown locale did not fall back to English")
	}
}
