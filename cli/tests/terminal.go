package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/opentable/sous/cli"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
	"github.com/xrash/smetrics"
)

type (
	// Terminal is a test harness for the CLI, providing easy
	// introspection into its inputs and outputs.
	Terminal struct {
		*cli.CLI
		Stdout, Stderr, Combined TestOutput
		History                  []string
		T                        *testing.T
		Graph                    psyringe.TestPsyringe
	}
	// TestOutput allows inspection of output streams from the Terminal.
	TestOutput struct {
		Name   string
		Buffer *bytes.Buffer
		T      *testing.T
	}
)

// NewTerminal creates a new test terminal.
func NewTerminal(t *testing.T, vstr string) *Terminal {
	v := semv.MustParse(vstr)
	baseout := TestOutput{"stdout", &bytes.Buffer{}, t}
	baseerr := TestOutput{"stderr", &bytes.Buffer{}, t}
	combined := TestOutput{"combined output", &bytes.Buffer{}, t}

	in := &bytes.Buffer{}
	out := io.MultiWriter(baseout.Buffer, combined.Buffer)
	err := io.MultiWriter(baseerr.Buffer, combined.Buffer)

	s := &cli.Sous{Version: v}
	di := graph.BuildTestGraph(v, in, out, err)
	ls, _ := logging.NewLogSinkSpy()
	c, er := cli.NewSousCLI(di, s, ls, out, err)
	if er != nil {
		panic(er)
	}

	testGraph := psyringe.TestPsyringe{Psyringe: di.Psyringe}
	return &Terminal{
		CLI:      c,
		Stdout:   baseout,
		Stderr:   baseerr,
		Combined: combined,
		History:  []string{},
		T:        t,
		Graph:    testGraph,
	}
}

// RunCommand takes a command line, turns it into args, and passes it to a CLI
// which is pre-populated with a fresh *cli.Sous command, OutWriter and
// ErrWriter, both mapped to Outputs for interrogation.
//
// Note: This cannot cope with arguments containing spaces, even if they are
// surrounded by quotes. We should add this feature if we need it.
func (t *Terminal) RunCommand(commandline string) {
	args := strings.Split(commandline, " ")
	os.Args[0] = args[0]
	t.CLI.Invoke(args)
	rr := fmt.Sprintf("shell> %s\n%s", commandline, t.Combined)
	if !strings.HasSuffix(rr, "\n") {
		rr += "<MISSING TRAILING NEWLINE>"
	}
	t.History = append(t.History, rr)
}

// Summary prints a summary of the session.
func (t *Terminal) Summary() string {
	buf := &bytes.Buffer{}
	for _, h := range t.History {
		buf.WriteString(h + "\n")
	}
	return buf.String()
}

// PrintFailureSummary prints the entire terminal session transcript if the
// failed. Otherwise, it does nothing.
func (t *Terminal) PrintFailureSummary() {
	if t.T.Failed() {
		t.T.Logf("Terminal Session Summary:\n%s", t.Summary())
	}
}

func (out TestOutput) String() string { return out.Buffer.String() }

// Lines returns a slice of strings representing lines in the output.
func (out TestOutput) Lines() []string { return strings.Split(out.String(), "\n") }

// LinesContaining is similar to Lines, but filters lines based on if they have
// a substring matching s.
func (out TestOutput) LinesContaining(s string) []string {
	lines := []string{}
	for _, l := range out.Lines() {
		if strings.Contains(l, s) {
			lines = append(lines, l)
		}
	}
	return lines
}

// NumLines returns the number of lines in the output.
func (out TestOutput) NumLines() int {
	return strings.Count(out.String(), "\n")
}

// HasLineMatching returns true if one of the output lines exactly matches s.
func (out TestOutput) HasLineMatching(s string) bool {
	for _, l := range out.Lines() {
		if l == s {
			return true
		}
	}
	return false
}

// ShouldBeEmpty fails the test if the output is not empty.
func (out TestOutput) ShouldBeEmpty() {
	if out.Buffer.Len() != 0 {
		out.T.Errorf("got length %d; want 0", out.Buffer.Len())
	}
}

// ShouldContain fails the test if the output does not contain byteSlice.
func (out TestOutput) ShouldContain(byteSlice []byte) {
	if !bytes.Contains(out.Buffer.Bytes(), byteSlice) {
		out.T.Errorf("did not contain %q", byteSlice)
	}
}

// ShouldContainString fails the test if the output does not contain s.
func (out TestOutput) ShouldContainString(s string) {
	if !strings.Contains(out.Buffer.String(), s) {
		out.T.Errorf("did not contain %q", s)
	}
}

// ShouldHaveExactLine fails the test if the output did not contain the line s.
func (out TestOutput) ShouldHaveExactLine(s string) {
	if out.HasLineMatching(s) {
		return
	}
	hint := out.similarLineHint(s)
	out.T.Errorf("expected %s to have exact line %q%s", out.Name, s, hint)
}

// ShouldHaveLineContaining is similar to ShouldHaveExactLine but only requires
// that the line contain the substring s rather then being equal to s.
func (out TestOutput) ShouldHaveLineContaining(s string) {
	for _, line := range out.Lines() {
		if strings.Contains(line, s) {
			return
		}
	}
	hint := out.similarLineHint(s)
	out.T.Errorf("expected %s to have line containing %q%s", out.Name, s, hint)
}

// ShouldHaveNumLines fails the test if the output does not have expected lines.
func (out TestOutput) ShouldHaveNumLines(expected int) {
	actual := out.NumLines()
	if actual == expected {
		return
	}
	out.T.Errorf("%s has %d lines; want %d", out.Name, actual, expected)
	if !strings.HasSuffix(out.String(), "\n") {
		out.T.Logf("MISSING TRAILING NEWLINE")
	}
}

func (out TestOutput) similarLineHint(s string) string {
	similar, i, goodMatch := out.MostSimilarLineTo(s)
	if !goodMatch {
		return ""
	}
	// we 1-index command output, "line 1" makes more sense than "line 0"
	i++
	return fmt.Sprintf("\nHowever, line %d looks similar: %q", i, similar)
}

// MostSimilarLineTo returns the most similar line in the output to the given
// string, if any of them have a JaroWinkler score >0.1. It returns the string
// (or empty), the index of that line, and a bool indicating if the score was
// greater than 0.1
func (out TestOutput) MostSimilarLineTo(s string) (
	winner string, index int, goodMatch bool) {
	index = -1
	if s == "" {
		return
	}
	max := 0.0
	for i, l := range out.Lines() {
		score := smetrics.JaroWinkler(l, s, 0.7, 4)
		if score > max {
			winner = l
			index = i
			max = score
		}
	}
	return winner, index, max > 0.1
}
