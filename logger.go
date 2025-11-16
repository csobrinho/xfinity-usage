package main

import (
	"fmt"

	log "github.com/google/logger"
)

type logger struct{ prefix string }

func (l *logger) bindArgs(v ...any) string {
	if len(v) == 0 {
		return ""
	}
	s := " "
	for i := 0; i < len(v); i += 2 {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%v: %v", v[i], v[i+1])
	}
	return s
}

// Satisfies the mqtt Logger interface.
func (l *logger) Println(v ...any) {
	log.Infof(l.prefix+"%v", fmt.Sprint(v...))
}

// Satisfies the mqtt Logger interface.
func (l *logger) Printf(format string, v ...any) {
	log.Infof(l.prefix+format, v...)
}

// Satisfies the retryablehttp.LeveledLogger interface.
func (l *logger) Error(msg string, v ...any) {
	log.Errorf(l.prefix+msg+"%s", l.bindArgs(v...))
}

// Satisfies the retryablehttp.LeveledLogger interface.
func (l *logger) Info(msg string, v ...any) {
	log.Infof(l.prefix+msg+"%s", l.bindArgs(v...))
}

// Satisfies the retryablehttp.LeveledLogger interface.
func (l *logger) Debug(msg string, v ...any) {
	log.V(1).Infof(l.prefix+msg+"%s", l.bindArgs(v...))
}

// Satisfies the retryablehttp.LeveledLogger interface.
func (l *logger) Warn(msg string, v ...any) {
	log.Warningf(l.prefix+msg+"%s", l.bindArgs(v...))
}
