package logging

import (
	"bytes"
	"runtime"
	"strings"
	"time"
)

// CallerInfo describes the source of a log message.
// It should be included in almost every message, to capture information about
// when and where the message was created.
type CallerInfo struct {
	callTime    time.Time
	goroutineID string
	callers     []uintptr
	excluding   []string
}

// GetCallerInfo captures a CallerInfo based on where it's called.
func GetCallerInfo(excluding ...string) CallerInfo {
	callers := make([]uintptr, 10)
	runtime.Callers(2, callers)

	prefixlen := len("goroutine ")
	buf := make([]byte, prefixlen+20)
	runtime.Stack(buf, false)
	buf = buf[prefixlen:]
	idx := bytes.IndexByte(buf, ' ')
	if idx != -1 {
		buf = buf[:idx]
	}
	return CallerInfo{
		callTime:    time.Now(),
		goroutineID: string(buf),
		callers:     callers,
		excluding:   excluding,
	}
}

// EachField calls f repeatedly with name/value pairs that capture what CallerInfo knows about the message.
func (info CallerInfo) EachField(f func(string, interface{})) {
	unknown := true
	frames := runtime.CallersFrames(info.callers)

	var frame runtime.Frame
	var more bool
FrameLoop:
	for frame, more = frames.Next(); more; frame, more = frames.Next() {
		for _, not := range append(info.excluding, "util/logging") {
			if strings.Index(frame.File, not) != -1 {
				continue FrameLoop
			}
		}
		unknown = false
		break
	}

	f("@timestamp", info.callTime.Format(time.RFC3339))
	f("thread-name", info.goroutineID)
	if unknown {
		f("file", "<unknown>")
		f("line", "<unknown>")
		f("function", "<unknown>")
		return
	}
	f("file", frame.File)
	f("line", frame.Line)
	f("function", frame.Function)
}
