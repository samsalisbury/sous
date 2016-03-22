package cli

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
		// isTerm reflects whether or not this output is connected to a terminal.
		isTerm bool
	}
)

func isTerm(w io.Writer) bool {
	file, isFile := w.(*os.File)
	return isFile && terminal.IsTerminal(int(file.Fd()))
}

func NewOutput(w io.Writer) Output {
	return Output{
		writer: w,
		isTerm: isTerm(w),
	}
}

func (o *Output) Writer() io.Writer {
	return o.writer
}

func (o *Output) Write(b []byte) {
	n, err := o.writer.Write(b)
	if err != nil {
		o.Errors = append(o.Errors, err)
	}
	if n != len(b) {
		e := fmt.Errorf("wrote only %d bytes of %d", n, len(b))
		o.Errors = append(o.Errors, e)
	}
}

func (o *Output) WriteString(s string) {
	o.Write([]byte(s))
}

func (o *Output) Println(v ...interface{}) {
	out := strings.Replace(fmt.Sprint(v...), "\n", "\n"+o.indent, -1)
	o.WriteString(o.indent + out + "\n")
}

func (o *Output) Printfln(format string, v ...interface{}) {
	o.Println(fmt.Sprintf(format, v...))
}

func (o *Output) SetIndentStyle(s string) {
	o.indentStyle = s
	o.setIndent()
}

func (o *Output) Indent() {
	o.indentSize++
	o.setIndent()
}

func (o *Output) Outdent() {
	if o.indentSize > 0 {
		o.indentSize--
		o.setIndent()
	}
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

func (o *Output) setIndent() {
	o.indent = strings.Repeat(o.indentStyle, o.indentSize)
}
