package logging

import (
	"fmt"
	"io"
	"log"
)

type Logger struct {
	inner *log.Logger
}

func New(w io.Writer) *Logger {
	return &Logger{inner: log.New(w, "", log.LstdFlags)}
}

func (l *Logger) Infof(format string, args ...any) {
	l.inner.Printf("[INFO] "+format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.inner.Printf("[WARN] "+format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.inner.Printf("[ERROR] "+format, args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.inner.Printf("[DEBUG] "+format, args...)
}

func (l *Logger) Printf(format string, args ...any) {
	l.inner.Printf(format, args...)
}

func (l *Logger) Stringf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
