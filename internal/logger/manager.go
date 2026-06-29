// Package logger is a pluggable, asynchronous logging system: storage
// (file or in-memory), severity filtering, and extension hooks are all
// swappable behind small interfaces. See DESIGN.md for the full rationale.
package logger

import (
	"log"
	"sync"
	"time"
)

// LogManager is the system's single entry point. Callers only ever talk to
// LogManager - they never touch a LogWriter/LogReader/LogHandler directly,
// which is what lets the storage backend and the set of registered
// handlers change without affecting anyone calling WriteLog/ReadLogs.
type LogManager struct {
	writer   LogWriter
	reader   LogReader
	minLevel LogLevel

	handlersMu sync.RWMutex
	handlers   []LogHandler

	entries chan LogEntry
	wg      sync.WaitGroup
}

// NewLogManager wires a storage backend in and starts the background
// goroutine that performs all actual writes, so WriteLog never blocks its
// caller on disk/network I/O.
func NewLogManager(writer LogWriter, reader LogReader, minLevel LogLevel) *LogManager {
	m := &LogManager{
		writer:   writer,
		reader:   reader,
		minLevel: minLevel,
		entries:  make(chan LogEntry, 256),
	}
	m.wg.Add(1)
	go m.run()
	return m
}

func (m *LogManager) run() {
	defer m.wg.Done()
	for entry := range m.entries {
		if err := m.writer.Write(entry); err != nil {
			log.Printf("logger: failed to write entry: %v", err)
		}

		m.handlersMu.RLock()
		handlers := m.handlers
		m.handlersMu.RUnlock()
		for _, h := range handlers {
			h.Handle(entry)
		}
	}
}

// WriteLog enqueues a new entry and returns immediately - the actual write
// happens asynchronously on the background goroutine started by
// NewLogManager. Entries below minLevel are dropped before they're even
// allocated as a LogEntry.
func (m *LogManager) WriteLog(level string, message string) {
	lvl := LogLevel(level)
	if !lvl.meets(m.minLevel) {
		return
	}
	m.entries <- NewLogEntry(lvl, message, nil)
}

// WriteLogFields is WriteLog plus structured key/value data, stored
// alongside the message for later filtering/analysis.
func (m *LogManager) WriteLogFields(level string, message string, fields map[string]interface{}) {
	lvl := LogLevel(level)
	if !lvl.meets(m.minLevel) {
		return
	}
	m.entries <- NewLogEntry(lvl, message, fields)
}

func (m *LogManager) ReadLogs(level string, filter LogFilter) []LogEntry {
	entries, err := m.reader.Read(LogLevel(level), filter)
	if err != nil {
		log.Printf("logger: failed to read entries: %v", err)
		return nil
	}
	return entries
}

func (m *LogManager) ClearLogs(before time.Time) error {
	return m.reader.Clear(before)
}

// RegisterLogHandler adds an extension point that gets called for every
// entry after it's written - see LogHandler's doc comment for examples.
func (m *LogManager) RegisterLogHandler(handler LogHandler) {
	m.handlersMu.Lock()
	defer m.handlersMu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// Close stops accepting new entries and blocks until every entry already
// queued has been written. Intended to be called during graceful shutdown,
// the same way ContainerService.Wait is.
func (m *LogManager) Close() {
	close(m.entries)
	m.wg.Wait()
}
