package logging

import (
	"bytes"
	"runtime"
	"strings"
)

// CallerInfo describes the source of a log message.
type CallerInfo struct {
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
		goroutineID: string(buf),
		callers:     callers,
		excluding:   excluding,
	}
}

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
