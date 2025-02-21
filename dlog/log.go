package dlog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	TRACE = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

var (
	TimeFormat  = "15:04:05"
	LogWriters  = []io.Writer{os.Stdout}
	LogChannels = []chan string{}
	LogLevel    = INFO
)

func RegisterLogChannel(c chan string) {
	LogChannels = append(LogChannels, c)
}

func _log(level int, format string, o ...any) {
	if level < LogLevel {
		return
	}
	builder := strings.Builder{}
	if len(TimeFormat) > 0 {
		builder.WriteString(time.Now().Format(TimeFormat) + " ")
	}
	l := fmt.Sprintf(format, o...)
	builder.WriteString(l)
	builder.WriteString("\n")
	writeLog(builder.String())
}

func writeLog(log string) {
	if LogWriters != nil {
		for _, w := range LogWriters {
			_, err := w.Write([]byte(log))
			if err != nil {
				panic(err)
			}
		}
	}
	if LogChannels != nil {
		go func() {
			for _, ch := range LogChannels {
				if ch != nil {
					ch <- log
				}
			}
		}()
	}
}

func Trace(format string, o ...any) {
	_log(TRACE, format, o...)
}
func Debug(format string, o ...any) {
	_log(DEBUG, format, o...)
}
func Info(format string, o ...any) {
	_log(INFO, format, o...)
}
func Warn(format string, o ...any) {
	_log(WARN, format, o...)
}
func Error(format string, o ...any) {
	_log(ERROR, format, o...)
}
func Fatal(format string, o ...any) {
	_log(FATAL, format, o...)
	os.Exit(1)
}
