package logger

import "time"

// LogEntry is the data shape every storage backend reads and writes.
// It's an interface (not a struct) so the same LogManager/LogWriter/
// LogReader contracts work regardless of how a particular entry chooses
// to represent itself internally.
type LogEntry interface {
	Level() LogLevel
	Message() string
	Timestamp() time.Time
	Fields() map[string]interface{}
}

type logEntry struct {
	level     LogLevel
	message   string
	timestamp time.Time
	fields    map[string]interface{}
}

func NewLogEntry(level LogLevel, message string, fields map[string]interface{}) LogEntry {
	return &logEntry{
		level:     level,
		message:   message,
		timestamp: time.Now(),
		fields:    fields,
	}
}

func (e *logEntry) Level() LogLevel                { return e.level }
func (e *logEntry) Message() string                { return e.message }
func (e *logEntry) Timestamp() time.Time           { return e.timestamp }
func (e *logEntry) Fields() map[string]interface{} { return e.fields }
