package logger_test

import (
	"fmt"

	"github.com/Franciswann/aidms-backend/internal/logger"
)

// alertOnError is a sample LogHandler showing the extension point in
// action: it doesn't touch LogManager or LogWriter at all, it just reacts
// to entries that already flow through the system.
type alertOnError struct{ count int }

func (a *alertOnError) Handle(entry logger.LogEntry) {
	if entry.Level() == logger.LevelError {
		a.count++
		fmt.Printf("ALERT: %d error(s) seen so far, latest: %s\n", a.count, entry.Message())
	}
}

func Example() {
	// Pick a storage backend - InMemoryLogStore here, but FileLogStore
	// (logger.NewFileLogStore("/var/log/aidms/app.log")) implements the
	// exact same LogWriter/LogReader pair and is a drop-in swap.
	store := logger.NewInMemoryLogStore()
	manager := logger.NewLogManager(store, store, logger.LevelInfo)

	alerts := &alertOnError{}
	manager.RegisterLogHandler(alerts)

	manager.WriteLog("info", "server started")
	manager.WriteLogFields("error", "failed to connect to database", map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	})

	// Close blocks until the background writer has drained everything
	// queued above - without it, ReadLogs below could race the writer.
	manager.Close()

	entries := manager.ReadLogs("error", logger.LogFilter{})
	fmt.Printf("stored error entries: %d\n", len(entries))

	// Output:
	// ALERT: 1 error(s) seen so far, latest: failed to connect to database
	// stored error entries: 1
}
