package tests

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/opentable/sous/cli"
	"github.com/opentable/sous/util/cmdr"
	"github.com/xrash/smetrics"
)

type (
	// Terminal is a test harness for the CLI, providing easy
	// introspection into its inputs and outputs.
	Terminal struct {
		*cmdr.CLI
		Stdout, Stderr, Combined TestOutput
		History                  []string
		T                        *testing.T
	}
	// Output allows inspection of output streams from the Terminal.
	TestOutput struct {
		Name   string
		Buffer *bytes.Buffer
		T      *testing.T
	}
)

func NewTerminal(t *testing.T, root cmdr.Command) *Terminal {
	out := TestOutput{"stdout", &bytes.Buffer{}, t}
	errout := TestOutput{"stderr", &bytes.Buffer{}, t}
	combined := TestOutput{"combined output", &bytes.Buffer{}, t}
	s := &cli.Sous{}
	c := &cmdr.CLI{
		Root: root,
		Out:  cmdr.NewOutput(io.MultiWriter(out.Buffer, combined.Buffer)),
		Err:  cmdr.NewOutput(io.MultiWriter(errout.Buffer, combined.Buffer)),
	}
	g, err := cli.BuildGraph(s, c)
	if err != nil {
		t.Fatal(err)
	}
	c.Hooks.PreExecute = func(c cmdr.Command) error {
		return g.Inject(c)
	}
	return &Terminal{c, out, errout, combined, []string{}, t}
}

// RunCommand takes a command line, turns it into args, and passes it to a CLI
// which is pre-populated with a fresh *cli.Sous command, OutWriter and
// ErrWriter, both mapped to Outputs for interrogation.
//
// Note: This cannot cope with arguments containing spaces, even if they are
// surrounded by quotes. We should add this feature if we need it.
func (t *Terminal) RunCommand(commandline string) {
	args := strings.Split(commandline, " ")
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

func (t *Terminal) PrintFailureSummary() {
	if t.T.Failed() {
		t.T.Logf("Terminal Session Summary:\n%s", t.Summary())
	}
}

func (out TestOutput) String() string { return out.Buffer.String() }

func (out TestOutput) Lines() []string { return strings.Split(out.String(), "\n") }

func (out TestOutput) LinesContaining(s string) []string {
	lines := []string{}
	for _, l := range out.Lines() {
		if strings.Contains(l, s) {
			lines = append(lines, l)
		}
	}
	return lines
}

func (out TestOutput) NumLines() int {
	return strings.Count(out.String(), "\n")
}

func (out TestOutput) HasLineMatching(s string) bool {
	for _, l := range out.Lines() {
		if l == s {
			return true
		}
	}
	return false
}

func (out TestOutput) ShouldHaveExactLine(s string) {
	if out.HasLineMatching(s) {
		return
	}
	hint := out.similarLineHint(s)
	out.T.Errorf("expected %s to have exact line %q%s", out.Name, s, hint)
}

func (out TestOutput) ShouldHaveLineContaining(s string) {
	for _, line := range out.Lines() {
		if strings.Contains(line, s) {
			return
		}
	}
	hint := out.similarLineHint(s)
	out.T.Errorf("expected %s to have line containing %q%s", out.Name, s, hint)
}

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
