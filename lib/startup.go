package sous

import "fmt"

type Startup struct {
	CheckReadyURIPath    string `yaml:",omitempty"`
	CheckReadyURITimeout int    `yaml:",omitempty"`
	Timeout              int    `yaml:",omitempty"`
}

var zeroStartup = Startup{}

// MergeDefaults merges default values with a Startup and returns the result
func (s Startup) MergeDefaults(base Startup) Startup {
	n := base
	if n.CheckReadyURIPath == zeroStartup.CheckReadyURIPath {
		n.CheckReadyURIPath = s.CheckReadyURIPath
	}

	if n.CheckReadyURITimeout == zeroStartup.CheckReadyURITimeout {
		n.CheckReadyURITimeout = s.CheckReadyURITimeout
	}

	if n.Timeout == zeroStartup.Timeout {
		n.Timeout = s.Timeout
	}

	return n
}

// UnmergeDefaults unmerges default values from a Startup based on an old value and returns the result
func (s Startup) UnmergeDefaults(base, old Startup) Startup {
	n := base

	if base.CheckReadyURIPath == s.CheckReadyURIPath &&
		old.CheckReadyURIPath == zeroStartup.CheckReadyURIPath {
		n.CheckReadyURIPath = zeroStartup.CheckReadyURIPath
	}

	if base.CheckReadyURITimeout == s.CheckReadyURITimeout &&
		old.CheckReadyURITimeout == zeroStartup.CheckReadyURITimeout {
		n.CheckReadyURITimeout = zeroStartup.CheckReadyURITimeout
	}

	if base.Timeout == s.Timeout &&
		old.Timeout == zeroStartup.Timeout {
		n.Timeout = zeroStartup.Timeout
	}

	return n
}

// Equal returns true if s == o.
func (s Startup) Equal(o Startup) bool {
	return len(s.diff(o)) == 0
}

func (s Startup) diff(o Startup) []string {
	diffs := []string{}
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }

	if s.CheckReadyURIPath != o.CheckReadyURIPath {
		diff("CheckReadyURIPath; this %q, other %q", s.CheckReadyURIPath, o.CheckReadyURIPath)
	}

	if s.CheckReadyURITimeout != o.CheckReadyURITimeout {
		diff("CheckReadyURITimeout; this %d, other %d", s.CheckReadyURITimeout, o.CheckReadyURITimeout)
	}

	if s.Timeout != o.Timeout {
		diff("Timeout; this %d, other %d", s.Timeout, o.Timeout)
	}

	return diffs
}
