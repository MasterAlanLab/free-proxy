// Package logging provides a slog-based logger that fans out to daily JSON log
// files and an in-process queryable store, mirroring the former JsonLogStore.
package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Entry is a single stored log line, matching the former JSON schema.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Module    string `json:"module"`
	Message   string `json:"message"`
}

// Store persists log entries to per-day files under logsDir and answers queries.
type Store struct {
	logsDir     string
	mu          sync.Mutex
	lastCleanup time.Time
}

// NewStore creates the logs directory and returns a Store.
func NewStore(logsDir string) (*Store, error) {
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, err
	}
	return &Store{logsDir: logsDir}, nil
}

func (s *Store) write(level, module, message string, t time.Time) {
	entry := Entry{
		Timestamp: t.Local().Format("2006-01-02 15:04:05"),
		Level:     level,
		Module:    module,
		Message:   message,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := filepath.Join(s.logsDir, t.Local().Format("2006-01-02")+".json")
	s.mu.Lock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		_, _ = f.Write(append(line, '\n'))
		_ = f.Close()
	}
	s.mu.Unlock()
	s.cleanup(t)
}

// Read returns stored entries for a date (default today), filtered by level and
// module substring, returning at most limit most-recent entries.
func (s *Store) Read(date, level, module string, limit int) []Entry {
	if date == "" {
		date = time.Now().Local().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return []Entry{}
	}
	path := filepath.Join(s.logsDir, date+".json")
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(path)
	if err != nil {
		return []Entry{}
	}
	defer f.Close()
	var out []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Entry
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		if level != "" && !strings.EqualFold(e.Level, level) {
			continue
		}
		if module != "" && !strings.Contains(strings.ToLower(e.Module), strings.ToLower(module)) {
			continue
		}
		out = append(out, e)
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func (s *Store) cleanup(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.lastCleanup.IsZero() && now.Sub(s.lastCleanup) < time.Hour {
		return
	}
	s.lastCleanup = now
	cutoff := now.Local().AddDate(0, 0, -3)
	entries, err := os.ReadDir(s.logsDir)
	if err != nil {
		return
	}
	for _, de := range entries {
		name := de.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		day, err := time.ParseInLocation("2006-01-02", strings.TrimSuffix(name, ".json"), time.Local)
		if err != nil {
			continue
		}
		if !day.After(cutoff.Truncate(24 * time.Hour)) {
			_ = os.Remove(filepath.Join(s.logsDir, name))
		}
	}
}

// storeHandler is a slog.Handler that writes each record into the Store while
// delegating formatted stderr output to a wrapped handler.
type storeHandler struct {
	store *Store
	next  slog.Handler
	attrs []slog.Attr
	group string
}

func (h *storeHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.next.Enabled(ctx, l)
}

func (h *storeHandler) Handle(ctx context.Context, r slog.Record) error {
	module := "free_proxy"
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "module" {
			module = a.Value.String()
			return false
		}
		return true
	})
	h.store.write(strings.ToUpper(r.Level.String()), module, r.Message, r.Time)
	return h.next.Handle(ctx, r)
}

func (h *storeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &storeHandler{store: h.store, next: h.next.WithAttrs(attrs), group: h.group}
}

func (h *storeHandler) WithGroup(name string) slog.Handler {
	return &storeHandler{store: h.store, next: h.next.WithGroup(name), group: name}
}

// Configure installs a slog default logger that persists to the Store and also
// emits human-readable lines to stderr. Returns the Store for querying.
func Configure(logsDir string) (*Store, error) {
	store, err := NewStore(logsDir)
	if err != nil {
		return nil, err
	}
	text := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(&storeHandler{store: store, next: text}))
	return store, nil
}
