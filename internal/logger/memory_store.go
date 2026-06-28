package logger

import (
	"sync"
	"time"
)

// InMemoryLogStore is a second LogWriter/LogReader implementation, used by
// tests (no filesystem needed) and as a concrete demonstration that
// LogManager really doesn't care which storage backend it's pointed at.
type InMemoryLogStore struct {
	mu      sync.Mutex
	entries []LogEntry
}

var (
	_ LogWriter = (*InMemoryLogStore)(nil)
	_ LogReader = (*InMemoryLogStore)(nil)
)

func NewInMemoryLogStore() *InMemoryLogStore {
	return &InMemoryLogStore{}
}

func (s *InMemoryLogStore) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	return nil
}

func (s *InMemoryLogStore) Read(level LogLevel, filter LogFilter) ([]LogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []LogEntry
	for _, e := range s.entries {
		if level != "" && e.Level() != level {
			continue
		}
		if !filter.Since.IsZero() && e.Timestamp().Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && e.Timestamp().After(filter.Until) {
			continue
		}
		out = append(out, e)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *InMemoryLogStore) Clear(before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	kept := s.entries[:0]
	for _, e := range s.entries {
		if e.Timestamp().Before(before) {
			continue
		}
		kept = append(kept, e)
	}
	s.entries = kept
	return nil
}
