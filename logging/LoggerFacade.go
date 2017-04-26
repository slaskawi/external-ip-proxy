package logging

import (
	"os"
	log "github.com/op/go-logging"
)

type Logger struct {
	Package string
	log *log.Logger
}

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info LogLevel = "info"
	Warning LogLevel = "warning"
	Error LogLevel = "error"
)

var LoggingFromLevel LogLevel = Info

func NewLogger(module string) (*Logger) {
	loggerInstance := &Logger{Package: module}
	loggerInstance.log = log.MustGetLogger(module)
	return loggerInstance
}

func init() {
	loggerFormat := log.MustStringFormatter(`%{color}%{time:15:04:05.000} %{module: 15s} %{shortfunc:7s} ▶ %{message}`)
	loggerBackend := log.NewLogBackend(os.Stdout, "", 0)
	loggerFormattedBackend := log.NewBackendFormatter(loggerBackend, loggerFormat)
	log.SetBackend(loggerFormattedBackend)
}

func (l *Logger) Info(format string, args ...interface{}) {
	if LoggingFromLevel == Warning || LoggingFromLevel == Error {
		return
	}
	if len(args) == 0 {
		l.log.Info(format)
	} else {
		l.log.Infof(format, args)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if LoggingFromLevel == Info || LoggingFromLevel == Warning || LoggingFromLevel == Error {
		return
	}
	if len(args) == 0 {
		l.log.Debug(format)
	} else {
		l.log.Debugf(format, args)
	}
}

func (l *Logger) Warning(format string, args ...interface{}) {
	if LoggingFromLevel == Error {
		return
	}
	if len(args) == 0 {
		l.log.Warning(format)
	} else {
		l.log.Warningf(format, args)
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	if len(args) == 0 {
		l.log.Error(format)
	} else {
		l.log.Errorf(format, args)
	}
}

