// Package logging is the server's one reading of P2KB_LOG_LEVEL. Every log
// line goes to the standard logger (stderr; stdout carries the MCP protocol)
// through a level check here.
package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
)

// Level orders log lines; a line is written when its level is at or above the
// configured one.
type Level int32

// Levels, lowest first.
const (
	Debug Level = iota
	Info
	Warn
	Error
)

// LevelEnv names the environment variable that sets the level.
const LevelEnv = "P2KB_LOG_LEVEL"

// DefaultLevel applies when P2KB_LOG_LEVEL is unset or unrecognised.
const DefaultLevel = Warn

var level atomic.Int32

func init() {
	level.Store(int32(ParseLevel(os.Getenv(LevelEnv))))
}

// ParseLevel maps "debug", "info", "warn" or "error" (any case) to a Level;
// anything else is DefaultLevel.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return Debug
	case "info":
		return Info
	case "warn", "warning":
		return Warn
	case "error":
		return Error
	}
	return DefaultLevel
}

// SetLevel sets the level and returns the previous one, so a test can restore it.
func SetLevel(l Level) Level { return Level(level.Swap(int32(l))) }

// Enabled reports whether lines at l are written.
func Enabled(l Level) bool { return l >= Level(level.Load()) }

// Debugf logs at Debug.
func Debugf(format string, args ...any) { logf(Debug, format, args...) }

// Infof logs at Info.
func Infof(format string, args ...any) { logf(Info, format, args...) }

// Warnf logs at Warn.
func Warnf(format string, args ...any) { logf(Warn, format, args...) }

// logf writes the line through log.Output so the logger's file:line flag
// names the caller of Debugf/Infof/Warnf, not this file.
func logf(l Level, format string, args ...any) {
	if Enabled(l) {
		_ = log.Output(3, fmt.Sprintf(format, args...))
	}
}
