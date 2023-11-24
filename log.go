// Package log is a logging subsystem that provides code optional location tracing and semi-automated subsystem registration and output control.
package log

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gookit/color"
	"go.uber.org/atomic"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// The Level settings used in proc
const (
	Off Level = iota
	Fatal
	Error
	Check
	Warn
	Info
	Debug
	Trace
)

// gLS is a helper to make more compact declarations of LevelSpec names and
// colors by using the Level LvlStr map.
func gLS(lvl Level, r, g, b byte) LevelSpec {
	return LevelSpec{
		Name:      LvlStr[lvl],
		Colorizer: color.Bit24(r, g, b, false).Sprintf,
	}
}

var (
	// LevelSpecs specifies the id, string name and color-printing function
	LevelSpecs = map[Level]LevelSpec{
		Off:   gLS(Off, 0, 0, 0),
		Fatal: gLS(Fatal, 255, 0, 0),
		Error: gLS(Error, 255, 128, 0),
		Check: gLS(Check, 255, 255, 0),
		Warn:  gLS(Warn, 128, 255, 0),
		Info:  gLS(Info, 0, 255, 0),
		Debug: gLS(Debug, 0, 128, 255),
		Trace: gLS(Trace, 128, 0, 255),
	}

	// LvlStr is a map that provides the uniform width strings that are printed
	// to identify the Level of a log entry.
	LvlStr = LevelMap{
		Off:   "off  ",
		Fatal: "fatal",
		Error: "error",
		Warn:  "warn ",
		Info:  "info ",
		Check: "check",
		Debug: "debug",
		Trace: "trace",
	}
	// log is your generic Logger creation invocation that uses the version data
	// in version.go that provides the current compilation path prefix for making
	// relative paths for log printing code locations.
	lvlStrs = map[string]Level{
		"off":   Off,
		"fatal": Fatal,
		"error": Error,
		"check": Check,
		"warn":  Warn,
		"info":  Info,
		"debug": Debug,
		"trace": Trace,
	}
	timeStampFormat           = "2006-01-02T15:04:05.000000000Z07:00"
	tty             io.Writer = os.Stderr
	writer                    = tty
	writerMx        sync.Mutex
	logLevel        = Info
	// App is the name of the application. Change this at the beginning of
	// an application main.
	App atomic.String
)

type (
	LevelMap map[Level]string
	// Level is a code representing a scale of importance and context for log
	// entries.
	Level int32
	// Println prints lists of interfaces with spaces in between
	Println func(a ...interface{})
	// Printf prints like fmt.Println surrounded by log details
	Printf func(format string, a ...interface{})
	// Prints  prints a spew.Sdump for an interface slice
	Prints func(a ...interface{})
	// Printc accepts a function so that the extra computation can be avoided if
	// it is not being viewed
	Printc func(closure func() string)
	// Chk is a shortcut for printing if there is an error, or returning true
	Chk func(e error) bool
	// LevelPrinter defines a set of terminal printing primitives that output
	// with extra data, time, level, and code location
	LevelPrinter struct {
		Ln Println
		// F prints like fmt.Println surrounded by log details
		F Printf
		// S uses spew.dump to show the content of a variable
		S Prints
		// C accepts a function so that the extra computation can be avoided if
		// it is not being viewed
		C Printc
		// Chk is a shortcut for printing if there is an error, or returning
		// true
		Chk Chk
	}
	// LevelSpec is a key pair of log level and the text colorizer used
	// for it.
	LevelSpec struct {
		Name      string
		Colorizer func(format string, a ...interface{}) string
	}
	// Logger is a set of log printers for the various Level items.
	Logger struct {
		F, E, W, I, D, T LevelPrinter
	}
)

func GetLevelByString(lvl string, def Level) (ll Level) {
	var exists bool
	if ll, exists = lvlStrs[lvl]; !exists {
		return def
	}
	return ll
}

func GetLevelName(ll Level) string {
	return strings.TrimSpace(LvlStr[ll])
}

// GetLoc calls runtime.Caller to get the path of the calling source code file.
func GetLoc(skip int) (output string) {
	_, file, line, _ := runtime.Caller(skip)
	output = fmt.Sprint(file, ":", line)
	return
}

// GetLogger returns a set of LevelPrinter with their subsystem preloaded
func GetLogger() (l *Logger) {
	return &Logger{
		getOnePrinter(Fatal),
		getOnePrinter(Error),
		getOnePrinter(Warn),
		getOnePrinter(Info),
		getOnePrinter(Debug),
		getOnePrinter(Trace),
	}
}

func SetLogLevel(l Level) {
	writerMx.Lock()
	defer writerMx.Unlock()
	logLevel = l
}

func GetLogLevel() (l Level) {
	writerMx.Lock()
	defer writerMx.Unlock()
	l = logLevel
	return
}

// SetTimeStampFormat sets a custom timeStampFormat for the logger
func SetTimeStampFormat(format string) {
	timeStampFormat = format
}

func (l LevelMap) String() (s string) {
	ss := make([]string, len(l))
	for i := range l {
		ss[i] = strings.TrimSpace(l[i])
	}
	return strings.Join(ss, " ")
}

func _c(level Level) Printc {
	return func(closure func() string) {
		logPrint(level, closure)()
	}
}
func _chk(level Level) Chk {
	return func(e error) (is bool) {
		if e != nil {
			logPrint(level,
				joinStrings(
					" ",
					"CHECK:",
					e,
				))()
			is = true
		}
		return
	}
}

func _f(level Level) Printf {
	return func(format string, a ...interface{}) {
		logPrint(
			level, func() string {
				return fmt.Sprintf(format, a...)
			},
		)()
	}
}

// The collection of the different types of log print functions,
// includes spew.Dump, closure and error check printers.

func _ln(l Level) Println {
	return func(a ...interface{}) {
		logPrint(l, joinStrings(" ", a...))()
	}
}
func _s(level Level) Prints {
	return func(a ...interface{}) {
		text := "spew:\n"
		if s, ok := a[0].(string); ok {
			text = strings.TrimSpace(s) + "\n"
			a = a[1:]
		}
		logPrint(
			level, func() string {
				return text + spew.Sdump(a...)
			},
		)()
	}
}

func getOnePrinter(level Level) LevelPrinter {
	return LevelPrinter{
		Ln:  _ln(level),
		F:   _f(level),
		S:   _s(level),
		C:   _c(level),
		Chk: _chk(level),
	}
}

// getTimeText is a helper that returns the current time with the
// timeStampFormat that is configured.
func getTimeText(tsf string) string { return time.Now().Format(tsf) }

// joinStrings constructs a string from a slice of interface same as Println but
// without the terminal newline
func joinStrings(sep string, a ...interface{}) func() (o string) {
	return func() (o string) {
		for i := range a {
			o += fmt.Sprint(a[i])
			if i < len(a)-1 {
				o += sep
			}
		}
		return
	}
}

// logPrint is the generic log printing function that provides the base
// format for log entries.
func logPrint(
	level Level,
	printFunc func() string,
) func() {
	return func() {
		writerMx.Lock()
		defer writerMx.Unlock()
		if level > logLevel {
			return
		}
		timeText := getTimeText(timeStampFormat)
		var loc string
		loc = GetLoc(3)
		formatString := "%s %s %s %s %s"
		var app string
		if len(App.Load()) > 0 {
			app = App.Load()
		}
		s := fmt.Sprintf(
			formatString,
			timeText,
			strings.ToUpper(app),
			LevelSpecs[level].Colorizer(
				LvlStr[level],
			),
			printFunc(),
			loc,
		)
		s = strings.TrimSuffix(s, "\n")
		_, _ = fmt.Fprintln(writer, s)
	}
}
