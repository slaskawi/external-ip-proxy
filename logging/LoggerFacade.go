package logging

import (
	"os"
	log "github.com/Sirupsen/logrus"
)

type Logger struct {
	Package string
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)

}

func (l *Logger) Info(format string, args ...interface{}) {
	log.Infof(format, args)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	log.Debug(format, args)
}

func (l *Logger) Warning(format string, args ...interface{}) {
	log.Warnf(format, args)
}

func (l *Logger) Error(format string, args ...interface{}) {
	log.Errorf(format, args)
}

