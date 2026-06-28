package logger

import "time"

// LogFilter narrows a Read down by time range and/or result count. An empty
// LogFilter (all zero values) means "no filtering beyond the level".
type LogFilter struct {
	Since time.Time
	Until time.Time
	Limit int
}

// LogReader is the pluggable point for "how do we query/prune what's
// already stored". It's paired with whatever LogWriter wrote the entries -
// the two are typically implemented by the same backend (e.g. FileLogStore
// implements both for the file backend) since they share the same storage.
type LogReader interface {
	// Read returns entries matching level (empty string = any level) and
	// filter, newest concerns aside - ordering is "as stored", which for
	// every backend here is write order.
	Read(level LogLevel, filter LogFilter) ([]LogEntry, error)

	// Clear permanently removes entries older than before.
	Clear(before time.Time) error
}
