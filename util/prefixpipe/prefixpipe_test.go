package prefixpipe

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPrefixPipe_ok_noconfig(t *testing.T) {

	dest := &bytes.Buffer{}

	pipe, err := New(dest, "prefix1: ")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fmt.Fprintln(pipe, "line one"); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(pipe, "line two"); err != nil {
		t.Fatal(err)
	}

	pipe.Close()
	pipe.Wait()

	want := "prefix1: line one\nprefix1: line two\n"

	got := dest.String()

	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}
