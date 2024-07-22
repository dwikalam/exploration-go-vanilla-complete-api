package logger

import (
	"io"
	"log"
)

type baseLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
}

func NewBaseLogger(stdout, stderr io.Writer) (baseLogger, error) {
	const (
		ResetColor  = "\033[0m"
		RedColor    = "\033[31m"
		YellowColor = "\033[33m"
		GreenColor  = "\033[32m"
	)

	var (
		infoLoggerPrefix  = GreenColor + "INFO: " + ResetColor
		warnLoggerPrefix  = YellowColor + "WARNING: " + ResetColor
		errorLoggerPrefix = RedColor + "ERROR: " + ResetColor
	)

	return baseLogger{
		infoLogger:  newLogger(stdout, infoLoggerPrefix),
		warnLogger:  newLogger(stdout, warnLoggerPrefix),
		errorLogger: newLogger(stderr, errorLoggerPrefix),
	}, nil
}

func (l *baseLogger) Info(message string) {
	l.infoLogger.Output(2, message)
}

func (l *baseLogger) Warn(message string) {
	l.warnLogger.Output(2, message)
}

func (l *baseLogger) Error(message string) {
	l.errorLogger.Output(2, message)
}

func newLogger(out io.Writer, prefix string) *log.Logger {
	return log.New(out, prefix, log.LstdFlags|log.Lshortfile)
}
