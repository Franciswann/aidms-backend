package logger

type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// severity gives each level a rank so WriteLog can filter out anything
// below the LogManager's configured minimum level. Unknown levels rank
// below everything, so a typo'd level string is filtered rather than
// silently accepted.
var severity = map[LogLevel]int{
	LevelDebug: 0,
	LevelInfo:  1,
	LevelWarn:  2,
	LevelError: 3,
}

func (l LogLevel) meets(min LogLevel) bool {
	return severity[l] >= severity[min]
}
