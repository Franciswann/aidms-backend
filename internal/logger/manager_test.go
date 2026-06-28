package logger

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogManager_WriteAndReadLogs(t *testing.T) {
	store := NewInMemoryLogStore()
	m := NewLogManager(store, store, LevelDebug)

	m.WriteLog("info", "server started")
	m.WriteLog("error", "db connection failed")
	m.Close() // blocks until both entries above are actually written

	all := m.ReadLogs("", LogFilter{})
	require.Len(t, all, 2)

	errorsOnly := m.ReadLogs("error", LogFilter{})
	require.Len(t, errorsOnly, 1)
	assert.Equal(t, "db connection failed", errorsOnly[0].Message())
}

func TestLogManager_LevelFiltering(t *testing.T) {
	store := NewInMemoryLogStore()
	// minLevel = warn: debug/info entries should be dropped before they're
	// ever handed to the writer.
	m := NewLogManager(store, store, LevelWarn)

	m.WriteLog("debug", "verbose detail")
	m.WriteLog("info", "routine event")
	m.WriteLog("error", "something broke")
	m.Close()

	all := m.ReadLogs("", LogFilter{})
	require.Len(t, all, 1)
	assert.Equal(t, "something broke", all[0].Message())
}

func TestLogManager_PreservesWriteOrder(t *testing.T) {
	store := NewInMemoryLogStore()
	m := NewLogManager(store, store, LevelDebug)

	const n = 200
	for i := 0; i < n; i++ {
		m.WriteLog("info", fmt.Sprintf("event-%d", i))
	}
	m.Close()

	got := m.ReadLogs("", LogFilter{})
	require.Len(t, got, n)
	for i, entry := range got {
		assert.Equal(t, fmt.Sprintf("event-%d", i), entry.Message())
	}
}

func TestLogManager_ClearLogs(t *testing.T) {
	store := NewInMemoryLogStore()
	m := NewLogManager(store, store, LevelDebug)

	m.WriteLog("info", "old entry")
	m.Close()

	cutoff := time.Now().Add(time.Hour) // after the entry above
	require.NoError(t, m.ClearLogs(cutoff))

	assert.Empty(t, m.ReadLogs("", LogFilter{}))
}

func TestLogManager_RegisterLogHandler(t *testing.T) {
	store := NewInMemoryLogStore()
	m := NewLogManager(store, store, LevelDebug)

	var handled []string
	m.RegisterLogHandler(handlerFunc(func(e LogEntry) {
		handled = append(handled, e.Message())
	}))

	m.WriteLog("info", "hello")
	m.Close()

	assert.Equal(t, []string{"hello"}, handled)
}

// handlerFunc adapts a plain func to LogHandler, the same way
// http.HandlerFunc adapts a func to http.Handler.
type handlerFunc func(LogEntry)

func (f handlerFunc) Handle(e LogEntry) { f(e) }

func TestFileLogStore_SameBehaviorAsInMemory(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileLogStore(dir + "/app.log")
	require.NoError(t, err)

	m := NewLogManager(store, store, LevelDebug)
	m.WriteLog("warn", "disk almost full")
	m.Close()

	got := m.ReadLogs("warn", LogFilter{})
	require.Len(t, got, 1)
	assert.Equal(t, "disk almost full", got[0].Message())
}
