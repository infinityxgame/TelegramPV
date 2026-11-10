package logbus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEmitNeutralizesLineBreaks(t *testing.T) {
	var out strings.Builder
	b := NewBus(16)
	b.mirror = &out

	e := b.Emit(LevelInfo, CatSystem,
		"arranque ok\r\n[ERROR] [FAKE] inyectado",
		"detalle\r\n[WARN] [FAKE] otro")

	if n := strings.Count(out.String(), "\n"); n != 1 {
		t.Fatalf("la salida debe tener exactamente 1 salto de línea, tiene %d: %q", n, out.String())
	}
	if strings.ContainsRune(out.String(), '\r') {
		t.Errorf("la salida contiene CR sin normalizar: %q", out.String())
	}
	if strings.ContainsAny(e.Message, "\r\n") || strings.ContainsAny(e.Detail, "\r\n") {
		t.Errorf("la entrada en memoria aún contiene saltos: message=%q detail=%q", e.Message, e.Detail)
	}
	if !strings.Contains(e.Message, `\n[ERROR] [FAKE] inyectado`) {
		t.Errorf("el texto debe conservarse con el salto neutralizado: %q", e.Message)
	}
}

func TestSanitizeLogField(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a\r\nb", `a\nb`},
		{"a\nb", `a\nb`},
		{"a\rb", `a\nb`},
		{"sin saltos", "sin saltos"},
	}
	for _, c := range cases {
		if got := sanitizeLogField(c.in); got != c.want {
			t.Errorf("sanitizeLogField(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDailyLogFile(t *testing.T) {
	tempDir := t.TempDir()
	b := NewBus(16)

	if err := b.OpenFile(tempDir); err != nil {
		t.Fatalf("OpenFile falló: %v", err)
	}

	b.Emit(LevelInfo, CatSystem, "prueba daily log", "detalle de prueba")
	b.CloseFile()

	today := time.Now().Format("02-01-06")
	expectedName := "telegrampv_" + today + ".log"
	expectedPath := filepath.Join(tempDir, "logs", expectedName)

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("no se pudo leer el archivo de log esperado %s: %v", expectedPath, err)
	}

	if !strings.Contains(string(data), "prueba daily log") {
		t.Errorf("el contenido del log no contiene el mensaje esperado: %s", string(data))
	}
}
