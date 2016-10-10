package config

import (
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
