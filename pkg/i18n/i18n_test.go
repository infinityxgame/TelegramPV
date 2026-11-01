package i18n

import (
	"testing"
)

func TestSetAndGetLanguage(t *testing.T) {
	SetLanguage("es")
	if CurrentLanguage() != "es" {
		t.Fatalf("expected es, got %s", CurrentLanguage())
	}

	SetLanguage("en")
	if CurrentLanguage() != "en" {
		t.Fatalf("expected en, got %s", CurrentLanguage())
	}
}

func TestTTranslation(t *testing.T) {
	SetLanguage("en")
	str := T("system.appStarted", "1.0.0")
	if str != "TelegramPV v1.0.0 started" {
		t.Fatalf("unexpected translation result: %s", str)
	}

	SetLanguage("es")
	strEs := T("system.appStarted", "1.0.0")
	if strEs != "TelegramPV v1.0.0 iniciado" {
		t.Fatalf("unexpected translation result: %s", strEs)
	}
}
