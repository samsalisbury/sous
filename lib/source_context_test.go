package sous

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert := assert.New(t)

	sc := SourceContext{
		OffsetDir:      "sub",
		RemoteURL:      "github.com/opentable/test",
		NearestTagName: "1.2.3",
		NearestTag:     Tag{Name: "1.2.3"},
	}
	id := sc.Version()
	assert.Equal("github.com/opentable/test", id.Location.Repo)
	assert.Equal("sub", string(id.Location.Dir))
	assert.Equal("1.2.3", id.Version.String())
}

func TestPrefixedVersion(t *testing.T) {
	assert := assert.New(t)

	sc := SourceContext{
		OffsetDir:      "sub",
		RemoteURL:      "github.com/opentable/test",
		NearestTagName: "release-1.2.3",
		NearestTag:     Tag{Name: "release-1.2.3"},
	}
	id := sc.Version()
	assert.Equal("github.com/opentable/test", id.Location.Repo)
	assert.Equal("sub", string(id.Location.Dir))
	assert.Equal("1.2.3", id.Version.String())
}

func TestNormalisedOffset_nosymlinks(t *testing.T) {
	rootDir := os.TempDir()
	rootDir = filepath.Join("tempDir", "TestNormalisedOffset_nosymlinks")
	if err := os.RemoveAll(rootDir); err != nil {
		t.Fatal(err)
	}
	offsetDir := filepath.Join(rootDir, "some-offset")

	os.MkdirAll(rootDir, 0777)
	os.MkdirAll(offsetDir, 0777)

	actual, err := NormalizedOffset(rootDir, offsetDir)
	if err != nil {
		t.Fatal(err)
	}
	expected := "some-offset"
	if actual != expected {
		t.Errorf("got %q; want %q", actual, expected)
	}
}

func TestParseSemverTagWithOptionalPrefix_happy(t *testing.T) {

	cases := []struct {
		in, wantPrefix, wantVersion string
	}{
		// OK
		{"1", "", "1.0.0"},
		{"1.2", "", "1.2.0"},
		{"1.2.3", "", "1.2.3"},
		{"1.2.3-pre1", "", "1.2.3-pre1"},
		{"1.2.3-pre1+meta1", "", "1.2.3-pre1+meta1"},
		{"prefix.a-1.2.3-pre1+meta1", "prefix.a-", "1.2.3-pre1+meta1"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			pre, v, err := parseSemverTagWithOptionalPrefix(c.in)
			if pre != c.wantPrefix {
				t.Errorf("got pre=%q; want %q", pre, c.wantPrefix)
			}
			gotVersion := v.Format(semv.Complete)
			if gotVersion != c.wantVersion {
				t.Errorf("got v=%q; want %q", gotVersion, c.wantVersion)
			}
			if err != nil {
				t.Errorf("got err=%q; want nil", err)
			}
		})
	}

}

func TestParseSemverTagWithOptionalPrefix_sad(t *testing.T) {

	cases := []struct {
		in, wantPrefix, wantErr string
	}{
		// Errors
		{"1_", "",
			`parsing semver version tag "1_": unexpected character '_' at position 1`},
		{"1.2-bad_pre", "",
			`parsing semver version tag "1.2-bad_pre": unexpected character '_' at position 7`},
		{"1.2_3", "",
			`parsing semver version tag "1.2_3": unexpected character '_' at position 3`},
		{"1.2.3-pre1§", "",
			`parsing semver version tag "1.2.3-pre1§": unexpected character '§' at position 10`},
		{"1.2.3-pre1+bad_meta1", "",
			`parsing semver version tag "1.2.3-pre1+bad_meta1": unexpected character '_' at position 14`},
		{"prefix.a-1.2.3-pre_1+meta1", "prefix.a-",
			`parsing semver version tag "1.2.3-pre_1+meta1" (ignoring the prefix "prefix.a-"): unexpected character '_' at position 9`},
	}
	// 1_: parsing semver version tag "1_": unexpected character '_' at position 1
	// 1.2.3-pre1§: parsing semver version tag "1.2.3-pre1§": unexpected character '§' at position 10
	// prefix.a-1.2.3-pre_1+meta1: parsing semver version tag "1.2.3-pre_1+meta1" (ignoring the prefix "prefix.a-"): unexpected character '_' at position 9
	// 1.2.3-pre1+bad_meta1: parsing semver version tag "1.2.3-pre1+bad_meta1": unexpected character '_' at position 14
	// 1.2_3: parsing semver version tag "1.2_3": unexpected character '_' at position 3
	// 1.2-bad_pre: parsing semver version tag "1.2-bad_pre": unexpected character '_' at position 7

	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			pre, v, err := parseSemverTagWithOptionalPrefix(c.in)
			if pre != c.wantPrefix {
				t.Errorf("got pre=%q; want %q", pre, c.wantPrefix)
			}
			if (v != semv.Version{}) {
				t.Errorf("got a non-zero version: %# v", v)
			}
			if err == nil {
				t.Fatalf("got nil error; want %q", c.wantErr)
			}
			gotErr := err.Error()
			if gotErr != c.wantErr {
				t.Errorf("got error %q; want %q", gotErr, c.wantErr)
				fmt.Fprintf(os.Stderr, "==> %s: %s\n", c.in, gotErr)
			}
		})
	}
}
