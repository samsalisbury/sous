package sous

import "fmt"

type Startup struct {
	CheckReadyURIPath    string `yaml:",omitempty"`
	CheckReadyURITimeout int    `yaml:",omitempty"`
	Timeout              int    `yaml:",omitempty"`
}

var zeroStartup = Startup{}

// MergeDefaults merges default values with a Startup and returns the result
func (s Startup) MergeDefaults(def Startup) Startup {
	n := s
	if n.CheckReadyURIPath == zeroStartup.CheckReadyURIPath {
		n.CheckReadyURIPath = def.CheckReadyURIPath
	}

	if n.CheckReadyURITimeout == zeroStartup.CheckReadyURITimeout {
		n.CheckReadyURITimeout = def.CheckReadyURITimeout
	}

	if n.Timeout == zeroStartup.Timeout {
		n.Timeout = def.Timeout
	}

	return n
}

// UnmergeDefaults unmerges default values from a Startup based on an old value and returns the result
func (s Startup) UnmergeDefaults(old, def Startup) Startup {
	n := s

	if s.CheckReadyURIPath == def.CheckReadyURIPath &&
		old.CheckReadyURIPath == zeroStartup.CheckReadyURIPath {
		n.CheckReadyURIPath = zeroStartup.CheckReadyURIPath
	}

	if s.CheckReadyURITimeout == def.CheckReadyURITimeout &&
		old.CheckReadyURITimeout == zeroStartup.CheckReadyURITimeout {
		n.CheckReadyURITimeout = zeroStartup.CheckReadyURITimeout
	}

	if s.Timeout == def.Timeout &&
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
