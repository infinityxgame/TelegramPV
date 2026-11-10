// Package logbus centraliza el registro de actividad de TelegramPV.
//
// Cumple tres funciones principales:
// 1. Captura todo lo que la aplicación escribe con el paquete estándar "log".
// 2. Ofrece Emit/Debug/Info/Success/Warn/Error para registrar eventos con contexto.
// 3. Mantiene las últimas entradas en memoria para la vista de logs en vivo del panel y las vuelca a archivo rotativo.
package logbus

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"telegrampv/pkg/i18n"
)

// Niveles de severidad. El panel los usa para colorear y filtrar.
const (
	LevelDebug   = "debug"
	LevelInfo    = "info"
	LevelSuccess = "success"
	LevelWarn    = "warn"
	LevelError   = "error"
)

// Categorías (origen de cada entrada de log).
const (
	CatSystem   = "System"
	CatTelegram = "Telegram"
	CatChats    = "Chats"
	CatSettings = "Settings"
	CatServer   = "Server"
	CatUpdater  = "Updater"
)

const (
	// DefaultCapacity define la capacidad máxima de entradas retenidas en memoria.
	DefaultCapacity = 2000
	// maxFileBytes es el tamaño a partir del cual se rota el archivo .log.
	maxFileBytes = 5 << 20 // 5 MiB
	// logDirName define la carpeta donde se guardan los archivos de registro.
	logDirName = "logs"
)

// Entry representa una línea normalizada del registro de actividad.
type Entry struct {
	ID       int64   `json:"id"`
	Time     float64 `json:"time"` // Segundos Unix con decimales
	Level    string  `json:"level"`
	Category string  `json:"category"`
	Message  string  `json:"message"`
	Detail   string  `json:"detail,omitempty"`
}

// Subscriber recibe cada entrada nueva en cuanto se registra.
type Subscriber func(Entry)

// Bus representa la estructura del registro de eventos.
type Bus struct {
	mu      sync.RWMutex
	entries []Entry
	nextID  int64
	max     int

	subsMu  sync.RWMutex
	subs    map[int]Subscriber
	nextSub int

	mirror io.Writer

	fileMu   sync.Mutex
	file     *os.File
	baseDir  string
	fileDate string
	filePath string
	fileSize int64
}

// NewBus crea un nuevo bus de logs con la capacidad indicada.
func NewBus(capacity int) *Bus {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Bus{
		entries: make([]Entry, 0, capacity),
		nextID:  1,
		max:     capacity,
		subs:    make(map[int]Subscriber),
		mirror:  os.Stderr,
	}
}

var std = NewBus(DefaultCapacity)

// Default devuelve el bus global.
func Default() *Bus { return std }

// OpenFile inicia el volcado de logs a disco en <dir>/logs/telegrampv_DD-MM-YY.log.
func (b *Bus) OpenFile(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("%s", i18n.T("system.logFileError"))
	}
	b.fileMu.Lock()
	defer b.fileMu.Unlock()

	b.baseDir = dir
	return b.openFileForDate(time.Now())
}

func (b *Bus) openFileForDate(t time.Time) error {
	if strings.TrimSpace(b.baseDir) == "" {
		return fmt.Errorf("%s", i18n.T("system.logFileError"))
	}
	dateStr := t.Format("02-01-06")
	if b.file != nil && b.fileDate == dateStr {
		return nil
	}

	if b.file != nil {
		_ = b.file.Close()
		b.file = nil
	}

	logDir := filepath.Join(b.baseDir, logDirName)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return err
	}
	fileName := fmt.Sprintf("telegrampv_%s.log", dateStr)
	path := filepath.Join(logDir, fileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	var size int64
	if fi, statErr := f.Stat(); statErr == nil {
		size = fi.Size()
	}

	b.file = f
	b.filePath = path
	b.fileDate = dateStr
	b.fileSize = size
	return nil
}

// FilePath devuelve la ruta del archivo de log activo.
func (b *Bus) FilePath() string {
	b.fileMu.Lock()
	defer b.fileMu.Unlock()
	return b.filePath
}

// CloseFile cierra el archivo de log si está abierto.
func (b *Bus) CloseFile() {
	b.fileMu.Lock()
	defer b.fileMu.Unlock()
	if b.file != nil {
		_ = b.file.Close()
		b.file = nil
	}
}

func (b *Bus) writeFile(line string) {
	b.writeFileForTime(time.Now(), line)
}

func (b *Bus) writeFileForTime(t time.Time, line string) {
	b.fileMu.Lock()
	defer b.fileMu.Unlock()

	if strings.TrimSpace(b.baseDir) == "" && b.file == nil {
		return
	}

	if err := b.openFileForDate(t); err != nil {
		return
	}

	n, err := b.file.WriteString(line + "\n")
	if err != nil {
		return
	}
	b.fileSize += int64(n)
	if b.fileSize < maxFileBytes {
		return
	}

	_ = b.file.Close()
	b.file = nil
	backup := b.filePath + ".1"
	_ = os.Remove(backup)
	if err := os.Rename(b.filePath, backup); err != nil {
		if f, oerr := os.OpenFile(b.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600); oerr == nil {
			b.file = f
			b.fileSize = 0
		}
		return
	}
	if f, oerr := os.OpenFile(b.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); oerr == nil {
		b.file = f
		b.fileSize = 0
	}
}

var logLineBreaks = strings.NewReplacer("\r\n", `\n`, "\n", `\n`, "\r", `\n`)

func sanitizeLogField(s string) string {
	return logLineBreaks.Replace(s)
}

// Emit registra un evento en el bus con nivel, categoría, mensaje y detalle.
func (b *Bus) Emit(level, category, message, detail string) Entry {
	level = normalizeLevel(level)
	if strings.TrimSpace(category) == "" {
		category = CatSystem
	}
	message = sanitizeLogField(strings.TrimSpace(message))
	detail = sanitizeLogField(strings.TrimSpace(detail))
	if message == "" && detail == "" {
		return Entry{}
	}

	now := time.Now()
	entry := Entry{
		Time:     float64(now.UnixNano()) / 1e9,
		Level:    level,
		Category: category,
		Message:  message,
		Detail:   detail,
	}

	b.mu.Lock()
	entry.ID = b.nextID
	b.nextID++
	b.entries = append(b.entries, entry)
	if len(b.entries) > b.max {
		n := copy(b.entries, b.entries[len(b.entries)-b.max:])
		b.entries = b.entries[:n]
	}
	b.mu.Unlock()

	formatted := formatLine(now, entry)
	if b.mirror != nil {
		_, _ = io.WriteString(b.mirror, formatted+"\n")
	}
	b.writeFileForTime(now, formatted)
	b.notify(entry)
	return entry
}

func (b *Bus) notify(entry Entry) {
	b.subsMu.RLock()
	subs := make([]Subscriber, 0, len(b.subs))
	for _, s := range b.subs {
		subs = append(subs, s)
	}
	b.subsMu.RUnlock()

	for _, s := range subs {
		func(fn Subscriber) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				line := fmt.Sprintf("panic in logbus subscriber: %v\n%s",
					r, strings.TrimRight(string(debug.Stack()), "\n"))
				fmt.Fprintln(os.Stderr, line)
				b.writeFile(line)
			}()
			fn(entry)
		}(s)
	}
}

// Subscribe registra un suscriptor de eventos de log.
func (b *Bus) Subscribe(fn Subscriber) func() {
	if fn == nil {
		return func() {}
	}
	b.subsMu.Lock()
	id := b.nextSub
	b.nextSub++
	b.subs[id] = fn
	b.subsMu.Unlock()

	return func() {
		b.subsMu.Lock()
		delete(b.subs, id)
		b.subsMu.Unlock()
	}
}

// Write implementa io.Writer para interceptar llamados a log.Printf.
func (b *Bus) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\r\n")
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		level, category, message := classifyRaw(line)
		b.Emit(level, category, message, "")
	}
	return len(p), nil
}

// Since devuelve las entradas posteriores a afterID.
func (b *Bus) Since(afterID int64, limit int) ([]Entry, int64) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > b.max {
		limit = b.max
	}
	start := sort.Search(len(b.entries), func(i int) bool {
		return b.entries[i].ID > afterID
	})
	slice := b.entries[start:]
	if len(slice) > limit {
		slice = slice[len(slice)-limit:]
	}
	out := make([]Entry, len(slice))
	copy(out, slice)
	return out, b.nextID - 1
}

// LastID devuelve el ID de la última entrada en memoria.
func (b *Bus) LastID() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nextID - 1
}

// Clear limpia el historial de entradas en memoria.
func (b *Bus) Clear() {
	b.mu.Lock()
	b.entries = b.entries[:0]
	b.mu.Unlock()
}

// ExportText convierte las entradas en texto plano.
func (b *Bus) ExportText() string {
	b.mu.RLock()
	entries := make([]Entry, len(b.entries))
	copy(entries, b.entries)
	b.mu.RUnlock()

	var sb strings.Builder
	for _, e := range entries {
		sec := int64(e.Time)
		nsec := int64((e.Time - float64(sec)) * 1e9)
		sb.WriteString(formatLine(time.Unix(sec, nsec), e))
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatLine(t time.Time, e Entry) string {
	line := fmt.Sprintf("%s %-7s %-13s %s",
		t.Format("2006-01-02 15:04:05"),
		strings.ToUpper(e.Level),
		"["+e.Category+"]",
		e.Message,
	)
	if e.Detail != "" {
		line += " — " + e.Detail
	}
	return line
}

func normalizeLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case LevelDebug:
		return LevelDebug
	case LevelSuccess:
		return LevelSuccess
	case LevelWarn, "warning":
		return LevelWarn
	case LevelError:
		return LevelError
	default:
		return LevelInfo
	}
}

var rawCategories = map[string]string{
	"SERVER":          CatServer,
	"TG CLIENT":       CatTelegram,
	"TG CLIENT ERROR": CatTelegram,
	"UPDATER":         CatUpdater,
	"SISTEMA":         CatSystem,
	"SYSTEM":          CatSystem,
	"CONFIG":          CatSettings,
	"SETTINGS":        CatSettings,
	"CHATS":           CatChats,
	"CHAT":            CatChats,
}

var (
	stdLogPrefix = regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?\s+`)
	tagPrefix    = regexp.MustCompile(`^\[([^\]]{1,40})\]\s*`)

	errorWords = regexp.MustCompile(`(?i)(error|falló|fallo|no se pudo|inválid|imposible|pánico|panic|rechazad|failed|failure|could not|cannot|unable|invalid|rejected|denied)`)
	warnWords  = regexp.MustCompile(`(?i)(advertencia|aviso|omitiendo|reintent|pausand|sin accesshash|sin sesión|no existe|no contiene|skipping|retr|paus|wait|missing|not found)`)
	okWords    = regexp.MustCompile(`(?i)(exitosa|establecida|completad|resuelto|actualizada|guardada|successful|established|completed|saved|resolved|updated)`)
	debugWords = regexp.MustCompile(`(?i)(solicitando|consultando|fetching|querying|requesting|reusing)`)
)

func classifyRaw(line string) (level, category, message string) {
	message = stdLogPrefix.ReplaceAllString(strings.TrimSpace(line), "")
	category = CatSystem

	if m := tagPrefix.FindStringSubmatch(message); m != nil {
		tag := strings.ToUpper(strings.TrimSpace(m[1]))
		if cat, ok := rawCategories[tag]; ok {
			category = cat
			message = strings.TrimSpace(message[len(m[0]):])
		}
		if strings.Contains(tag, "ERROR") {
			return LevelError, category, message
		}
	}

	switch {
	case errorWords.MatchString(message):
		level = LevelError
	case warnWords.MatchString(message):
		level = LevelWarn
	case debugWords.MatchString(message):
		level = LevelDebug
	case okWords.MatchString(message):
		level = LevelSuccess
	default:
		level = LevelInfo
	}
	return level, category, message
}

// ---------------------------------------------------------------------------
// API global de paquete
// ---------------------------------------------------------------------------

func Init(dir string) error { return std.OpenFile(dir) }

func Attach() {
	log.SetFlags(0)
	log.SetOutput(std)
}

func Detach() {
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stderr)
}

func Emit(level, category, message, detail string) Entry {
	return std.Emit(level, category, message, detail)
}

func Debug(category, message string, details ...string) Entry {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	return std.Emit(LevelDebug, category, message, det)
}

func Info(category, message string, details ...string) Entry {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	return std.Emit(LevelInfo, category, message, det)
}

func Success(category, message string, details ...string) Entry {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	return std.Emit(LevelSuccess, category, message, det)
}

func Warn(category, message string, details ...string) Entry {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	return std.Emit(LevelWarn, category, message, det)
}

func Error(category, message string, details ...string) Entry {
	det := ""
	if len(details) > 0 {
		det = details[0]
	}
	return std.Emit(LevelError, category, message, det)
}

func Subscribe(fn Subscriber) func()                  { return std.Subscribe(fn) }
func Since(afterID int64, limit int) ([]Entry, int64) { return std.Since(afterID, limit) }
func LastID() int64                                   { return std.LastID() }
func ExportText() string                              { return std.ExportText() }
func FilePath() string                                { return std.FilePath() }
func CloseFile()                                      { std.CloseFile() }

func GetEntries(sinceID int64) ([]Entry, int64) {
	return Since(sinceID, DefaultCapacity)
}

func Clear() {
	std.Clear()
	std.Emit(LevelInfo, CatSystem, i18n.T("system.logCleared"), "")
}
