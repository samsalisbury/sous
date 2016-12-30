package cmdr

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/opentable/sous/util/cmdr/style"

	"golang.org/x/crypto/ssh/terminal"
)

type (
	// Output is a convenience wrapper around an io.Writer that provides extra
	// features for formatting text, such as indentation and the ability to
	// emit tables. It is designed to be used sequentially, writing and changing
	// context on each call to one of its methods.
	Output struct {
		// Verbosity is the verbosity of this output.
		Verbosity Verbosity
		// Errors contains any errors this output has encountered whilst
		// writing to Writer.
		Errors []error
		// Style is the default style for this output. Note that styles are only
		// used when the output is connected to a terminal.
		Style      style.Style
		styleStack []style.Style
		// Writer is the io.Writer that this output writes to.
		writer io.Writer
		// indentSize is the number of times to repeat IndentStyle in the
		// current context.
		indentSize int
		// indentStyle is the string used for the current indent, it is repeated
		// indentSize times at the beginning of each line.
		indentStyle string
		// indent is the eagerly managed current indent string
		indent string
		// isTerm reflects whether or not this output is connected to a
		// terminal.
		isTerm bool
	}
)

func isTerm(w io.Writer) bool {
	file, isFile := w.(*os.File)
	return isFile && terminal.IsTerminal(int(file.Fd()))
}

// NewOutput creates a new Output, you may optionally pass any number of
// functions, each of which will be called on the Output before it is returned.
// You can use this to create and configure an output in a single statement.
func NewOutput(w io.Writer, configFunc ...func(*Output)) *Output {
	out := &Output{
		Style:       style.DefaultStyle(),
		indentStyle: DefaultIndentString,
		writer:      w,
		isTerm:      isTerm(w),
	}
	for _, f := range configFunc {
		f(out)
	}
	return out
}

func (o *Output) PushStyle(s style.Style) {
	o.styleStack = append(o.styleStack, o.Style)
	o.Style = s
}

func (o *Output) PopStyle() {
	l := len(o.styleStack)
	if l == 0 {
		return
	}
	i := l - 1
	o.Style = o.styleStack[i]
	o.styleStack = o.styleStack[:i]
}

func (o *Output) Write(b []byte) (int, error) {
	if o.isTerm && utf8.Valid(b) {
		fmt.Fprintf(o.writer, "\033[%sm", o.Style)
		defer fmt.Fprintf(o.writer, "\033[0m")
	}
	n, err := o.writer.Write(b)
	if err != nil {
		o.Errors = append(o.Errors, err)
	}
	if n != len(b) {
		e := fmt.Errorf("wrote only %d bytes of %d", n, len(b))
		o.Errors = append(o.Errors, e)
	}
	return n, err
}

func (o *Output) WriteString(s string) {
	o.Write([]byte(s))
}

// Println prints a line, respecting current indentation.
func (o *Output) Println(v ...interface{}) {
	out := strings.Replace(fmt.Sprint(v...), "\n", "\n"+o.indent, -1)
	o.WriteString(o.indent + out + "\n")
}

// Printfln is similar to Println, except it takes a format string.
func (o *Output) Printfln(format string, v ...interface{}) {
	o.Println(fmt.Sprintf(format, v...))
}

func (o *Output) Printf(format string, v ...interface{}) {
	fmt.Fprintf(o, format, v...)
}

func (o *Output) Table(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	colWidths := make([]int, len(rows[0]))
	for _, cells := range rows {
		for col, cell := range cells {
			if colWidths[col] < len(cell) {
				colWidths[col] = len(cell)
			}
		}
	}
	colFormats := make([]string, len(colWidths))
	for col, width := range colWidths {
		colFormats[col] = fmt.Sprintf("%%-%ds", width+2)
	}
	for _, cells := range rows {
		rowStr := ""
		for col, cell := range cells {
			rowStr += fmt.Sprintf(colFormats[col], cell)
		}
		o.Println(rowStr)
	}
}
