package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"
	"testing"
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
