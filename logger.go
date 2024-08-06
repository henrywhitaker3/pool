package pool

import (
	"fmt"
	"log"
)

var (
	DefaultLogger = &stdOutLogger{}
	NullLogger    = &nullLogger{}
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

type nullLogger struct{}

func (n *nullLogger) Debugf(format string, args ...any) {}
func (n *nullLogger) Errorf(format string, args ...any) {}
