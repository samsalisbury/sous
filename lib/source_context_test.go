package sous

import (
	"os"
	"path/filepath"
	"testing"

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
