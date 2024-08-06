package pool

import (
	"fmt"
	"log"
)

type Logger interface {
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)
}

type stdOutLogger struct{}

func (s *stdOutLogger) Debugf(format string, args ...any) {
	format = fmt.Sprintf("[DEBUG] - %s", format)
	log.Printf(format, args...)
}

func (s *stdOutLogger) Errorf(format string, args ...any) {
	format = fmt.Sprintf("[ERROR] - %s", format)
	log.Printf(format, args...)
}

func defaultLogger() Logger {
	return &stdOutLogger{}
}
