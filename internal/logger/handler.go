package logger

// LogHandler is the extension point for reacting to every log entry
// (forward it to a remote aggregator, count errors and page someone, feed
// a metrics system) by implementing this and registering with
// LogManager.RegisterLogHandler, without LogManager or LogWriter ever
// needing to change.
type LogHandler interface {
	Handle(entry LogEntry)
}
