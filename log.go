package logrotating

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	LOG_LEVEL_PANIC = iota
	LOG_LEVEL_FATAL
	LOG_LEVEL_ERROR
	LOG_LEVEL_WARNING
	LOG_LEVEL_INFO
	LOG_LEVEL_DEBUG
)

var levelArr = []string{
	"PANIC",
	"FATAL",
	"ERROR",
	"WARNING",
	"INFO",
	"DEBUG",
}

const (
	Ldate      = 1 << iota                  // the date in the local time zone: 2009-01-23
	Ltime                                   // the time in the local time zone: 01:23:23.456
	Llongfile                               // full file name and line number: /a/b/c/d.go:23
	Lshortfile                              // final file name element and line number: d.go:23. overrides Llongfile
	LstdFlags  = Ldate | Ltime | Lshortfile // initial values for the standard logger
)

type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	file  string
	size  int // 0代表无限大，文件大于字节数时，将文件重命名
	flag  int
	level int
}

func New(out io.Writer, file string, size int, flag int) *Logger {
	return &Logger{out: out, file: file, size: size, flag: flag, level: LOG_LEVEL_INFO}
}

var std = New(os.Stderr, "", 0, LstdFlags)

func SetLogger(logger *Logger) {
	std = logger
}

func SetLogLevel(level int) {
	std.SetLogLevel(level)
}

func Error(args ...interface{}) {
	std.print(2, LOG_LEVEL_ERROR, args...)
}

func Errorln(args ...interface{}) {
	std.println(2, LOG_LEVEL_ERROR, args...)
}

func Errorf(format string, args ...interface{}) {
	std.printf(2, LOG_LEVEL_ERROR, format, args...)
}

func (l *Logger) SetLogLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) print(calldepth int, level int, args ...interface{}) {
	if level <= l.level {
		l.output(calldepth+1, level, fmt.Sprint(args...))
	}
}

func (l *Logger) println(calldepth int, level int, args ...interface{}) {
	if level <= l.level {
		l.output(calldepth+1, level, fmt.Sprintln(args...))
	}
}

func (l *Logger) printf(calldepth int, level int, format string, args ...interface{}) {
	if level <= l.level {
		l.output(calldepth+1, level, fmt.Sprintf(format, args...))
	}
}

func (l *Logger) output(calldepth int, level int, s string) error {
	now := time.Now() // get this early.
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.flag&(Lshortfile) != 0 {
		// Release lock while getting caller info - it's expensive.
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.mu.Lock()
	}
	buf := []byte{}
	l.formatHeader(&buf, now, file, line, level)
	buf = append(buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf = append(buf, '\n')
	}

	l.checkFile(now)
	_, err := l.out.Write(buf)
	return err
}

func (l *Logger) checkFile(t time.Time) {
	if l.size <= 0 {
		return
	}
	f, err := os.Stat(l.file)
	if err != nil {
		return
	}
	if l.size >= f.Size() {
		timestamp = t.Unix()
		newFileName := fmt.Sprintf("%s.%lld", l.file, timestamp)
		os.Rename(l.file, newFileName)
	}
}

func (l *Logger) formatHeader(buf *[]byte, t time.Time, file string, line int, level int) {
	if l.flag&(Ldate|Ltime) != 0 {
		*buf = append(*buf, '[')
		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			*buf = append(*buf, dateStr...)
		}
		if l.flag&(Ltime) != 0 {
			hour, min, sec := t.Clock()
			microsecond := t.Nanosecond() / 1e6
			timeStr := fmt.Sprintf(" %02d:%02d:%02d.%03d", hour, min, sec, microsecond)
			*buf = append(*buf, timeStr...)
		}
		*buf = append(*buf, "] "...)
	}
	if l.flag&(Lshortfile|Llongfile) != 0 {
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		fileStr := fmt.Sprintf("[%s:%d] ", file, line)
		*buf = append(*buf, fileStr...)
	}
	*buf = append(*buf, levelArr[level]...)
	*buf = append(*buf, ' ')
}
