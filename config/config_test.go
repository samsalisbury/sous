package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"
	"testing"

	"github.com/opentable/sous/ext/docker"
)

func TestDefaultStateLocation(t *testing.T) {
	testCases := []struct {
		XDGDataHome, StateLocation string
	}{
		{"", "~/.local/share/sous/state"}, // Note: ~ is handled in code below.
		{"some/dir", "some/dir/sous/state"},
	}

	c := &Config{}
	for i, tc := range testCases {
		os.Setenv("XDG_DATA_HOME", tc.XDGDataHome)
		expected := tc.StateLocation
		u, err := user.Current()
		if err != nil {
			t.Error(err)
			continue
		}
		if expected[0] == '~' {
			expected = strings.TrimPrefix(expected, "~")
			expected = path.Join(u.HomeDir, expected)
		}
		actual, err := c.defaultStateLocation()
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%d: %q got %q; want %q", i, tc.XDGDataHome, actual, expected)
		}
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := &Config{
		Server:      "not_a_url",
		SiblingURLs: []string{"tcp://is.a.url"},
	}

	checkNotValid := func() {
		if err := cfg.Validate(); err == nil {
			t.Errorf("%v returns true from Validate()", cfg)
		}
	}

	checkValid := func() {
		if err := cfg.Validate(); err != nil {
			t.Errorf("%v returns false from Validate()", cfg)
		}
	}

	checkNotValid()

	cfg.Server = "https://a.sous.server"
	checkNotValid()

	cfg.SiblingURLs[0] = "http://sibling.sous"
	checkValid()

	cfg.Server = "zxcv"
	checkNotValid()

	cfg.Server = ""
	checkValid()
}

func TestConfig_Equals(t *testing.T) {
	expected := &Config{
		StateLocation: "statelocation",
		Server:        "server",
		SiblingURLs:   []string{"sibling", "urls"},
		BuildStateDir: "buildstatedir",
		Docker: docker.Config{
			RegistryHost:       "registryhost",
			DatabaseDriver:     "databasedriver",
			DatabaseConnection: "databaseconnection",
		},
	}
	var actual *Config

	checkNotEqual := func() {
		if expected.Equal(actual) {
			t.Errorf("expected.Equal(actual) was true:\n%v\n%v", expected, actual)
		}
		if actual.Equal(expected) {
			t.Errorf("actual.Equal(expected) was true:\n%v\n%v", actual, expected)
		}
	}

	actual = &Config{}
	checkNotEqual()

	actual.StateLocation = "statelocation"
	checkNotEqual()

	actual.Server = "server"
	checkNotEqual()

	actual.BuildStateDir = "buildstatedir"
	checkNotEqual()

	actual.Docker = expected.Docker
	checkNotEqual()

	actual.SiblingURLs = []string{"urls", "sibling"}
	checkNotEqual()

	actual.SiblingURLs = []string{"sibling", "urls"}
	if !expected.Equal(actual) {
		t.Errorf("expected.Equal(actual) was false:\n%v\n%v", expected, actual)
	}
	if !actual.Equal(expected) {
		t.Errorf("actual.Equal(expected) was false:\n%v\n%v", actual, expected)
	}

	// Confirming that the two Docker structs are separate memory
	actual.Docker.DatabaseConnection = ""
	checkNotEqual()
}

func TestEnsureDirExists(t *testing.T) {
	testDataDir := "testdata/gen"
	if err := os.RemoveAll(testDataDir); err != nil {
		t.Fatal(err)
	}
	extantDir := path.Join(testDataDir, "test-dir")
	if err := os.MkdirAll(extantDir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDirExists(extantDir); err != nil {
		t.Error(err)
	}
	nonExistentDir := path.Join(testDataDir, "other-dir")
	if err := EnsureDirExists(nonExistentDir); err != nil {
		t.Error(err)
	}
	s, err := os.Stat(nonExistentDir)
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsDir() {
		t.Errorf("created path %q is not a directory", nonExistentDir)
	}
	filePath := path.Join(testDataDir, "a-file")
	if err := ioutil.WriteFile(filePath, []byte("x"), 0777); err != nil {
		t.Fatal(err)
	}
	expectedErr := fmt.Sprintf("%q exists and is not a directory", filePath)
	err = EnsureDirExists(filePath)
	if err == nil {
		t.Errorf("got nil error; want %q", expectedErr)
	}
	actualErr := err.Error()
	if actualErr != expectedErr {
		t.Errorf("got error %q; want %q", actualErr, expectedErr)
	}
}
