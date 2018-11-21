package sous

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDumpDeployments(t *testing.T) {
	assert := assert.New(t)

	io := &bytes.Buffer{}
	ds := NewDeployments()
	ds.Add(&Deployment{ClusterName: "andromeda"})

	DumpDeployments(io, ds)
	assert.Regexp(`andromeda`, io.String())
}

func TestJSONDeployments(t *testing.T) {
	io := &bytes.Buffer{}
	ds := NewDeployments()
	ds.Add(&Deployment{SourceID: MustNewSourceID("andromeda", "", "1")})
	ds.Add(&Deployment{SourceID: MustNewSourceID("el_gordo", "", "1")})

	JSONDeployments(io, ds)
	var nonemptyLines = func(b []byte) [][]byte {
		return bytes.FieldsFunc(b, func(r rune) bool { return r == '\n' })
	}

	// Assert count.
	gotLines := nonemptyLines(io.Bytes())
	gotLineCount := len(gotLines)
	wantLineCount := 2
	if gotLineCount != wantLineCount {
		t.Fatalf("got %d lines; want %d", gotLineCount, wantLineCount)
	}

	// Assert valid JSON on each line.
	gotDeployments := make([]Deployment, gotLineCount)
	for i, line := range gotLines {
		d := Deployment{}
		if err := json.Unmarshal(line, &d); err != nil {
			// Just give up on the first invalid JSON.
			t.Fatalf("invalid JSON on line %d: %q", i, line)
		}
		gotDeployments[i] = d
	}

	// Assert deployments round-trip correctly.
	for _, got := range gotDeployments {
		original, ok := ds.Get(got.ID())
		if !ok {
			t.Errorf("got deployment not in original set: %v", got)
			continue
		}
		if !got.Equal(original) {
			t.Errorf("deployment %q did not round-trip correctly: got %v; want %v",
				got.ID(), got, original)
		}
	}

}
