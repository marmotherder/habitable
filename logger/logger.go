package logger

import (
	"fmt"
	"log"
	"os"
)

const (
	ERROR = 0
	WARN  = 1
	INFO  = 2
	DEBUG = 3
	TRACE = 4
)

type Logger interface {
	GetLevel() int
	Fatal(exitCode int, message interface{}, params ...interface{})
	Error(message interface{}, params ...interface{})
	Warn(message interface{}, params ...interface{})
	Info(message interface{}, params ...interface{})
	Debug(message interface{}, params ...interface{})
	Trace(message interface{}, params ...interface{})
}

type DefaultLogger struct {
	Level int
}

func (l DefaultLogger) GetLevel() int {
	return l.Level
}

func (l DefaultLogger) Fatal(exitCode int, message interface{}, params ...interface{}) {
	fmtPrint("FATAL", message, params...)
	os.Exit(exitCode)
}

func (l DefaultLogger) Error(message interface{}, params ...interface{}) {
	if l.Level >= ERROR {
		fmtPrint("ERROR", message, params...)
	}
}

func (l DefaultLogger) Warn(message interface{}, params ...interface{}) {
	if l.Level >= WARN {
		fmtPrint("WARN", message, params...)
	}
}

func (l DefaultLogger) Info(message interface{}, params ...interface{}) {
	if l.Level >= INFO {
		fmtPrint("INFO", message, params...)
	}
}

func (l DefaultLogger) Debug(message interface{}, params ...interface{}) {
	if l.Level >= DEBUG {
		fmtPrint("DEBUG", message, params...)
	}
}

func (l DefaultLogger) Trace(message interface{}, params ...interface{}) {
	if l.Level >= TRACE {
		fmtPrint("TRACE", message, params...)
	}
}

func fmtPrint(level string, message interface{}, params ...interface{}) {
	fmtMessage := ""
	if len(params) > 0 {
		if messageString, ok := message.(string); ok {
			fmtMessage = fmt.Sprintf(messageString, params...)
		} else {
			fmtMessage = "failed to print log message, was sent a fmt request, but message was not a string"
		}
	} else {
		fmtMessage = fmt.Sprint(message)
	}

	log.Printf(level+": %s\n", fmtMessage)
}
