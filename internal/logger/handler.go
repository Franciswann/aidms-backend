package logger

// LogHandler is the extension point from the PDF's "允許後續擴展" requirement.
// Anything that wants to react to every log entry (forward it to a remote
// aggregator, count errors and page someone, feed a metrics system) can
// implement this and register itself with LogManager.RegisterLogHandler,
// without LogManager or LogWriter ever needing to change.
type LogHandler interface {
	Handle(entry LogEntry)
}
