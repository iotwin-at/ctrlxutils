package ctrlxutils

import (
	"context"
	"fmt"
	"log/syslog"
	"time"

	"github.com/uoul/go-common/log"
)

type JournaldLogger struct {
	writer *syslog.Writer
	logLvl log.LogLevel
}

// Debug implements log.ILogger.
func (j *JournaldLogger) Debug(message string) {
	j.log(j.writer.Debug, message)
}

// Debugf implements log.ILogger.
func (j *JournaldLogger) Debugf(format string, a ...any) {
	j.log(j.writer.Debug, fmt.Sprintf(format, a...))
}

// Error implements log.ILogger.
func (j *JournaldLogger) Error(message string) {
	j.log(j.writer.Err, message)
}

// Errorf implements log.ILogger.
func (j *JournaldLogger) Errorf(format string, a ...any) {
	j.log(j.writer.Err, fmt.Sprintf(format, a...))
}

// Fatal implements log.ILogger.
func (j *JournaldLogger) Fatal(message string) {
	j.log(j.writer.Crit, message)
}

// Fatalf implements log.ILogger.
func (j *JournaldLogger) Fatalf(format string, a ...any) {
	j.log(j.writer.Crit, fmt.Sprintf(format, a...))
}

// Info implements log.ILogger.
func (j *JournaldLogger) Info(message string) {
	j.log(j.writer.Info, message)
}

// Infof implements log.ILogger.
func (j *JournaldLogger) Infof(format string, a ...any) {
	j.log(j.writer.Info, fmt.Sprintf(format, a...))
}

// Trace implements log.ILogger.
func (j *JournaldLogger) Trace(message string) {
	j.log(j.writer.Debug, message)
}

// Tracef implements log.ILogger.
func (j *JournaldLogger) Tracef(format string, a ...any) {
	j.log(j.writer.Debug, fmt.Sprintf(format, a...))
}

// Warning implements log.ILogger.
func (j *JournaldLogger) Warning(message string) {
	j.log(j.writer.Warning, message)
}

// Warningf implements log.ILogger.
func (j *JournaldLogger) Warningf(format string, a ...any) {
	j.log(j.writer.Warning, fmt.Sprintf(format, a...))
}

func (j *JournaldLogger) log(logfun func(string) error, msg string) {
	if err := logfun(msg); err != nil {
		fmt.Printf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), msg)
	}
}

func NewJournaldLogger(ctx context.Context, tag string, logLevel log.LogLevel) (log.ILogger, error) {
	writer, err := syslog.New(syslog.LOG_USER, tag)
	if err != nil {
		return nil, err
	}
	return &JournaldLogger{
		writer: writer,
		logLvl: logLevel,
	}, nil
}
