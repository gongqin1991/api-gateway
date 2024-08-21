package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Level int

func (l Level) ToString() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

type Logger struct {
	OutputDir string
	ErrorFile string
	Hostname  string

	date      string
	ErrorLog  *log.Logger
	internal  *log.Logger
	closeable []func()
	mu        sync.Mutex

	parent *Logger
	fields []string
}

const (
	_ Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

const (
	indexError = iota
	indexDate
)

var logger = &Logger{closeable: make([]func(), 2)}

func (l *Logger) root() *Logger {
	if l.parent == nil {
		return l
	}
	return l.parent.root()
}

func (l *Logger) Setup() error {
	l.OutputDir = viper.GetString("common.log.out-dir")
	l.Hostname = viper.GetString("hostname")
	l.ErrorFile = viper.GetString("common.log.error-file")
	err := os.Mkdir(l.OutputDir, 0777)
	if err != nil && strings.Contains(err.Error(), "file exists") {
		err = nil
	}
	if err != nil {
		return err
	}
	filename := filepath.Join(l.OutputDir, l.ErrorFile)
	log, closeFunc, err := newLogger(l.Hostname, filename)
	if err != nil {
		err = errors.Wrap(err, "new error log file error")
	} else {
		l.ErrorLog = log
		l.closeable[indexError] = closeFunc
	}
	return err
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	log := &Logger{parent: l}
	//request id
	rid := getRIDFromCtx(ctx)
	log.fields = []string{rid}
	return log
}

func (l *Logger) WithFields(fields ...string) *Logger {
	log := &Logger{parent: l}
	log.fields = fields
	return log
}

func (l *Logger) Info(args ...any) {
	l.levelPrintln(LevelInfo, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.levelPrintf(LevelInfo, format, args...)
}

func (l *Logger) Warn(args ...any) {
	l.levelPrintln(LevelWarn, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.levelPrintf(LevelWarn, format, args...)
}

func (l *Logger) Error(args ...any) {
	l.levelPrintln(LevelError, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.levelPrintf(LevelError, format, args...)
}

func (l *Logger) Fatal(args ...any) {
	if err := l.root().ErrorLog.Output(2, fmt.Sprintln(args...)); err != nil {
		fmt.Println("Fatal:", err)
	}
}

func (l *Logger) Fatalf(format string, args ...any) {
	if err := l.root().ErrorLog.Output(2, fmt.Sprintf(format, args...)); err != nil {
		fmt.Println("Fatalf:", err)
	}
}

func (l *Logger) levelPrintln(level Level, args ...any) {
	mayNewLogger(logger, time.Now())
	fields := collectFields(l)
	printArgc := len(args)
	printArgc += 1           //level
	printArgc += len(fields) //fields
	printArgs := make([]any, 0, printArgc)
	printArgs = append(printArgs, level.ToString()) //level
	for _, field := range fields {                  //fields
		printArgs = append(printArgs, field)
	}
	printArgs = append(printArgs, args...)
	if err := l.root().internal.Output(3, fmt.Sprintln(printArgs...)); err != nil {
		fmt.Println("levelPrintln:", err)
	}
}

func (l *Logger) levelPrintf(level Level, format string, args ...any) {
	mayNewLogger(logger, time.Now())
	fields := collectFields(l)
	stringer := strings.Builder{}
	printArgs := make([]any, 0)
	//level
	stringer.WriteString("%s")
	stringer.WriteRune(' ')
	printArgs = append(printArgs, level.ToString())
	//request id
	for _, field := range fields {
		stringer.WriteString("%s")
		stringer.WriteRune(' ')
		printArgs = append(printArgs, field)
	}
	stringer.WriteString(format)
	printArgs = append(printArgs, args...)
	if err := l.root().internal.Output(3, fmt.Sprintf(stringer.String(), printArgs...)); err != nil {
		fmt.Println("levelPrintf:", err)
	}
}

func collectFields(log *Logger) []string {
	logs := make([]*Logger, 0)
	fields := make([]string, 0)
	for log != nil {
		if log.parent != nil && len(log.fields) > 0 {
			logs = append(logs, log)
		}
		log = log.parent
	}

	for i := len(logs) - 1; i >= 0; i-- {
		l := logs[i]
		fields = append(fields, l.fields...)
	}
	return fields
}

func newLogger(prefix, filename string) (*log.Logger, func(), error) {
	fs, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, err
	}
	l := log.New(fs, prefix+" ", log.Lmsgprefix|log.LstdFlags|log.Lshortfile)
	return l, func() { _ = fs.Close() }, nil
}

func mayNewLogger(logger *Logger, now time.Time) {
	newLogger := func(date string) *log.Logger {
		filename := filepath.Join(logger.OutputDir, environment+date)
		log, closeFunc, err := newLogger(logger.Hostname, filename)
		if err != nil {
			fmt.Printf("create new log file error,filename:%s,err:%v\n", filename, err)
		} else {
			logger.closeable[indexDate] = closeFunc
		}
		return log
	}
	newDate := now.Format("20060102")
	logger.mu.Lock()
	oldDate := logger.date
	if newDate == oldDate {
		logger.mu.Unlock()
		return
	}

	defer logger.mu.Unlock()

	logger.date = newDate
	if close := logger.closeable[indexDate]; close != nil {
		close()
	}
	if logs := newLogger(newDate); logs != nil {
		logger.internal = logs
	} else {
		panic("new log file error")
	}

}

func (l *Logger) Destroy() {
	for _, close := range l.closeable {
		if close != nil {
			close()
		}
	}
}
