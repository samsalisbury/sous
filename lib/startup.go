package sous

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

	if s.CheckReadURITimeout == def.CheckReadURITimeout &&
		old.CheckReadURITimeout == zeroStartup.CheckReadURITimeout {
		n.CheckReadURITimeout = zeroStartup.CheckReadURITimeout
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

	if s.CheckReadyURIPath != nil {
		if o.CheckReadyURIPath == nil {
			diff("CheckReadyURIPath; this %q, other empty", *s.CheckReadyURIPath)
		} else if *s.CheckReadyURIPath != *o.CheckReadyURIPath {
			diff("CheckReadyURIPath; this %q, other %q", *s.CheckReadyURIPath, *o.CheckReadyURIPath)
		}
	} else {
		if o.CheckReadyURIPath != nil {
			diff("CheckReadyURIPath; this empty, other %q", *o.CheckReadyURIPath)
		}
	}

	if s.CheckReadyURITimeout != nil {
		if o.CheckReadyURITimeout == nil {
			diff("CheckReadyURITimeout; this %d, other empty", *s.CheckReadyURITimeout)
		} else if *s.CheckReadyURITimeout != *o.CheckReadyURITimeout {
			diff("CheckReadyURITimeout; this %d, other %d", *s.CheckReadyURITimeout, *o.CheckReadyURITimeout)
		}
	} else {
		if o.CheckReadyURITimeout != nil {
			diff("CheckReadyURITimeout; this empty, other %d", *o.CheckReadyURITimeout)
		}
	}

	if s.Timeout != nil {
		if o.Timeout == nil {
			diff("Timeout; this %d, other empty", *s.Timeout)
		} else if *s.Timeout != *o.Timeout {
			diff("Timeout; this %d, other %d", *s.Timeout, *o.Timeout)
		}
	} else {
		if o.Timeout != nil {
			diff("Timeout; this empty, other %d", *o.Timeout)
		}
	}

	return diffs
}
