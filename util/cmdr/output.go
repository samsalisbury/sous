package cmdr

import (
	"fmt"
	"io"
	"os"
	"strings"

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
		indentStyle: DefaultIndentString,
		writer:      w,
		isTerm:      isTerm(w),
	}
	for _, f := range configFunc {
		f(out)
	}
	return out
}

func (o *Output) Write(b []byte) (int, error) {
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
