package logging

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	for in, want := range map[string]Level{
		"debug": Debug, "DEBUG": Debug, " info ": Info, "warn": Warn, "warning": Warn,
		"error": Error, "": DefaultLevel, "verbose": DefaultLevel,
	} {
		if got := ParseLevel(in); got != want {
			t.Errorf("ParseLevel(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestDefaultLevelIsWarn(t *testing.T) {
	// Unset P2KB_LOG_LEVEL kept today's behaviour: warnings print, errors
	// returned to the client and the startup banner do not.
	if DefaultLevel != Warn {
		t.Errorf("DefaultLevel = %d, want Warn", DefaultLevel)
	}
}

func TestLinesAreGatedByLevel(t *testing.T) {
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	prev := SetLevel(Info)
	t.Cleanup(func() {
		SetLevel(prev)
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})

	Debugf("debug line")
	Infof("info line")
	Warnf("warn line")

	out := buf.String()
	if strings.Contains(out, "debug line") {
		t.Error("debug line written at Info")
	}
	if !strings.Contains(out, "info line") || !strings.Contains(out, "warn line") {
		t.Errorf("output = %q, want the info and warn lines", out)
	}
	if !Enabled(Error) || Enabled(Debug) {
		t.Error("Enabled disagrees with level Info")
	}
}
