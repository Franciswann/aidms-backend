package logger

// LogWriter is the pluggable point for "where do new log entries go".
// Different backends (file, database, a remote aggregator) satisfy this
// same interface so LogManager never needs to know which one it's using.
type LogWriter interface {
	Write(entry LogEntry) error
}
