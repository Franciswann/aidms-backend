package logger

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// jsonLogEntry is the on-disk shape: one of these, JSON-encoded, per line
// (JSON Lines format). This is what satisfies the PDF's "structured
// logging" requirement - each line is independently parseable JSON, which
// is what tools like jq, Elasticsearch/Loki ingestion, etc. expect.
type jsonLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// FileLogStore implements both LogWriter and LogReader against a single
// JSON-lines file. Read/Write/Clear all go through the same mutex because
// they share one file on disk - without it, a Clear rewriting the file
// concurrently with a Write appending to it would corrupt the file.
//
// Each Write opens, appends, and closes the file rather than holding a
// long-lived file handle open. That costs a few syscalls per entry, but it
// means Clear (which replaces the file's contents) never has to worry about
// invalidating a handle some other method is still holding - simplicity
// over raw throughput, which is the right tradeoff here since LogManager
// already serializes writes through a single background goroutine.
type FileLogStore struct {
	mu   sync.Mutex
	path string
}

var (
	_ LogWriter = (*FileLogStore)(nil)
	_ LogReader = (*FileLogStore)(nil)
)

func NewFileLogStore(path string) (*FileLogStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	return &FileLogStore{path: path}, nil
}

func (s *FileLogStore) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(jsonLogEntry{
		Timestamp: entry.Timestamp(),
		Level:     entry.Level(),
		Message:   entry.Message(),
		Fields:    entry.Fields(),
	})
	if err != nil {
		return err
	}
	line = append(line, '\n')
	_, err = f.Write(line)
	return err
}

func (s *FileLogStore) Read(level LogLevel, filter LogFilter) ([]LogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllLocked()
	if err != nil {
		return nil, err
	}

	var out []LogEntry
	for _, je := range all {
		if !matches(je, level, filter) {
			continue
		}
		out = append(out, NewLogEntry(je.Level, je.Message, je.Fields))
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *FileLogStore) Clear(before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllLocked()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, je := range all {
		if je.Timestamp.Before(before) {
			continue
		}
		line, err := json.Marshal(je)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return w.Flush()
}

func (s *FileLogStore) readAllLocked() ([]jsonLogEntry, error) {
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []jsonLogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var je jsonLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &je); err != nil {
			continue // skip a malformed line rather than failing the whole read
		}
		all = append(all, je)
	}
	return all, scanner.Err()
}

func matches(je jsonLogEntry, level LogLevel, filter LogFilter) bool {
	if level != "" && je.Level != level {
		return false
	}
	if !filter.Since.IsZero() && je.Timestamp.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && je.Timestamp.After(filter.Until) {
		return false
	}
	return true
}
