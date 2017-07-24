package sous

import (
	"fmt"
	"sort"
	"strings"
)

// Startup is the configuration for startup checks for a service deployment.
// c.f. DeployConfig for use.
type Startup struct { //                             Singularity fields
	SkipCheck bool `yaml:",omitempty"`

	ConnectDelay    int `yaml:",omitempty"` // Healthcheck.StartupDelaySeconds
	Timeout         int `yaml:",omitempty"` // Healthcheck.StartupTimeoutSeconds
	ConnectInterval int `yaml:",omitempty"` // Healthcheck.StartupIntervalSeconds

	CheckReadyProtocol        string `yaml:",omitempty"` // Healthcheck.Protocol
	CheckReadyURIPath         string `yaml:",omitempty"` // Healthcheck.URI
	CheckReadyPortIndex       int    `yaml:",omitempty"` // Healthcheck.PortIndex
	CheckReadyFailureStatuses []int  `yaml:",omitempty"` // Healthcheck.FailureStatusCodes
	CheckReadyURITimeout      int    `yaml:",omitempty"` // Healthcheck.ResponseTimeoutSeconds
	CheckReadyInterval        int    `yaml:",omitempty"` // Healthcheck.IntervalSeconds
	CheckReadyRetries         int    `yaml:",omitempty"` // Healthcheck.MaxRetries

	// ??? We don't deploy fixed port services...
	// ??? CheckReadyPortNumber int    `yaml:",omitempty"` // Healthcheck.PortNumber

	// XXX it would be possible to do a CheckReadyTimeout instead of MaxRetries...
}

var zeroStartup = Startup{}

// Validate implements Flawed on Startup.
func (s *Startup) Validate() []Flaw {
	flaws := []Flaw{}

	if !s.SkipCheck {
		if s.ConnectDelay < 0 {
			flaws = append(flaws, FatalFlaw("ConnectDelay less than zero: %d!", s.ConnectDelay))
		}
		if s.Timeout < 0 {
			flaws = append(flaws, FatalFlaw("Timeout less than zero: %d!", s.Timeout))
		}
		if s.ConnectInterval < 0 {
			flaws = append(flaws, FatalFlaw("ConnectInterval less than zero: %d!", s.ConnectInterval))
		}

		if s.CheckReadyPortIndex < 0 {
			flaws = append(flaws, FatalFlaw("CheckReadyPortIndex less than zero: %d!", s.CheckReadyPortIndex))
		}
		if s.CheckReadyURITimeout < 0 {
			flaws = append(flaws, FatalFlaw("CheckReadyURITimeout less than zero: %d!", s.CheckReadyURITimeout))
		}
		if s.CheckReadyInterval < 0 {
			flaws = append(flaws, FatalFlaw("CheckReadyInterval less than zero: %d!", s.CheckReadyInterval))
		}
		if s.CheckReadyRetries < 0 {
			flaws = append(flaws, FatalFlaw("CheckReadyRetries less than zero: %d!", s.CheckReadyRetries))
		}

		switch s.CheckReadyProtocol {
		default:
			flaws = append(flaws, FatalFlaw("CheckReadyProtocol must be HTTP or HTTPS, was %q.", s.CheckReadyProtocol))
		case "https", "http":
			flaws = append(flaws, NewFlaw(fmt.Sprintf("CheckReadyProtocol must be HTTP or HTTPS, was %q (lowercase).", s.CheckReadyProtocol),
				func() error {
					s.CheckReadyProtocol = strings.ToUpper(s.CheckReadyProtocol)
					return nil
				}))
		case "HTTPS", "HTTP":
		}

		for _, status := range s.CheckReadyFailureStatuses {
			if status < 0 {
				flaws = append(flaws, FatalFlaw("CheckReadyFailureStatuses includes a value less that zero: %d", status))
			}
		}
	}

	return flaws
}

// MergeDefaults merges default values with a Startup and returns the result
func (s Startup) MergeDefaults(base Startup) Startup {
	n := base

	if n.SkipCheck == zeroStartup.SkipCheck {
		n.SkipCheck = s.SkipCheck
	}

	if n.ConnectDelay == zeroStartup.ConnectDelay {
		n.ConnectDelay = s.ConnectDelay
	}

	if n.Timeout == zeroStartup.Timeout {
		n.Timeout = s.Timeout
	}

	if n.ConnectInterval == zeroStartup.ConnectInterval {
		n.ConnectInterval = s.ConnectInterval
	}

	if n.CheckReadyProtocol == zeroStartup.CheckReadyProtocol {
		n.CheckReadyProtocol = s.CheckReadyProtocol
	}

	if n.CheckReadyURIPath == zeroStartup.CheckReadyURIPath {
		n.CheckReadyURIPath = s.CheckReadyURIPath
	}

	if n.CheckReadyPortIndex == zeroStartup.CheckReadyPortIndex {
		n.CheckReadyPortIndex = s.CheckReadyPortIndex
	}

	// The zero slice length is zero, but leaving this for consitency.
	if len(n.CheckReadyFailureStatuses) == len(zeroStartup.CheckReadyFailureStatuses) {
		n.CheckReadyFailureStatuses = s.CheckReadyFailureStatuses
	}

	if n.CheckReadyURITimeout == zeroStartup.CheckReadyURITimeout {
		n.CheckReadyURITimeout = s.CheckReadyURITimeout
	}

	if n.CheckReadyInterval == zeroStartup.CheckReadyInterval {
		n.CheckReadyInterval = s.CheckReadyInterval
	}

	if n.CheckReadyRetries == zeroStartup.CheckReadyRetries {
		n.CheckReadyRetries = s.CheckReadyRetries
	}

	return n
}

// UnmergeDefaults unmerges default values from a Startup based on an old value and returns the result
func (s Startup) UnmergeDefaults(base, old Startup) Startup {
	n := base

	if base.SkipCheck == s.SkipCheck &&
		old.SkipCheck == zeroStartup.SkipCheck {
		n.SkipCheck = zeroStartup.SkipCheck
	}

	if base.ConnectDelay == s.ConnectDelay &&
		old.ConnectDelay == zeroStartup.ConnectDelay {
		n.ConnectDelay = zeroStartup.ConnectDelay
	}

	if base.Timeout == s.Timeout &&
		old.Timeout == zeroStartup.Timeout {
		n.Timeout = zeroStartup.Timeout
	}

	if base.ConnectInterval == s.ConnectInterval &&
		old.ConnectInterval == zeroStartup.ConnectInterval {
		n.ConnectInterval = zeroStartup.ConnectInterval
	}

	if base.CheckReadyProtocol == s.CheckReadyProtocol &&
		old.CheckReadyProtocol == zeroStartup.CheckReadyProtocol {
		n.CheckReadyProtocol = zeroStartup.CheckReadyProtocol
	}

	if base.CheckReadyURIPath == s.CheckReadyURIPath &&
		old.CheckReadyURIPath == zeroStartup.CheckReadyURIPath {
		n.CheckReadyURIPath = zeroStartup.CheckReadyURIPath
	}

	if base.CheckReadyPortIndex == s.CheckReadyPortIndex &&
		old.CheckReadyPortIndex == zeroStartup.CheckReadyPortIndex {
		n.CheckReadyPortIndex = zeroStartup.CheckReadyPortIndex
	}

	if len(base.CheckReadyFailureStatuses) == len(s.CheckReadyFailureStatuses) &&
		len(old.CheckReadyFailureStatuses) == len(zeroStartup.CheckReadyFailureStatuses) {
		sort.Ints(base.CheckReadyFailureStatuses)
		sort.Ints(s.CheckReadyFailureStatuses)
		equal := true
		for n, sfs := range s.CheckReadyFailureStatuses {
			if sfs != base.CheckReadyFailureStatuses[n] {
				equal = false
				break
			}
		}
		if equal {
			n.CheckReadyFailureStatuses = zeroStartup.CheckReadyFailureStatuses
		}
	}

	if base.CheckReadyURITimeout == s.CheckReadyURITimeout &&
		old.CheckReadyURITimeout == zeroStartup.CheckReadyURITimeout {
		n.CheckReadyURITimeout = zeroStartup.CheckReadyURITimeout
	}

	if base.CheckReadyInterval == s.CheckReadyInterval &&
		old.CheckReadyInterval == zeroStartup.CheckReadyInterval {
		n.CheckReadyInterval = zeroStartup.CheckReadyInterval
	}

	if base.CheckReadyRetries == s.CheckReadyRetries &&
		old.CheckReadyRetries == zeroStartup.CheckReadyRetries {
		n.CheckReadyRetries = zeroStartup.CheckReadyRetries
	}

	return n
}

// Equal returns true if s == o.
func (s Startup) Equal(o Startup) bool {
	return len(s.diff(o)) == 0
}

func (s Startup) diff(o Startup) []string {
	diffs := []string{}
	diff := func(format string, a ...interface{}) {
		d := fmt.Sprintf(format, a...)
		diffs = append(diffs, d)
	}

	l, r := s, o
	if s.SkipCheck == true { //redundant, but makes ConfirmTree happy
		l = zeroStartup
	}
	if o.SkipCheck == true {
		r = zeroStartup
	}

	if l.ConnectDelay != r.ConnectDelay {
		diff("ConnectDelay; this %d, other %d", l.ConnectDelay, r.ConnectDelay)
	}

	if l.ConnectInterval != r.ConnectInterval {
		diff("ConnectInterval; this %d, other %d", l.ConnectInterval, r.ConnectInterval)
	}

	if l.Timeout != r.Timeout {
		diff("Timeout; this %d, other %d", l.Timeout, r.Timeout)
	}

	if l.CheckReadyProtocol != r.CheckReadyProtocol {
		diff("CheckReadyProtocol; this %q, other %q", l.CheckReadyProtocol, r.CheckReadyProtocol)
	}

	if l.CheckReadyPortIndex != r.CheckReadyPortIndex {
		diff("CheckReadyPortIndex; this %d, other %d", l.CheckReadyPortIndex, r.CheckReadyPortIndex)
	}

	if len(l.CheckReadyFailureStatuses) != len(r.CheckReadyFailureStatuses) {
		diff("CheckReadyFailureStatuses; this %v, other %v", l.CheckReadyFailureStatuses, r.CheckReadyFailureStatuses)
	} else {
		sort.Ints(r.CheckReadyFailureStatuses)
		sort.Ints(l.CheckReadyFailureStatuses)
		for n, sfs := range l.CheckReadyFailureStatuses {
			if sfs != r.CheckReadyFailureStatuses[n] {
				diff("CheckReadyFailureStatuses; this %v, other %v", l.CheckReadyFailureStatuses, r.CheckReadyFailureStatuses)
				break
			}
		}
	}

	if l.CheckReadyInterval != r.CheckReadyInterval {
		diff("CheckReadyInterval; this %d, other %d", l.CheckReadyInterval, r.CheckReadyInterval)
	}

	if l.CheckReadyRetries != r.CheckReadyRetries {
		diff("CheckReadyRetries; this %d, other %d", l.CheckReadyRetries, r.CheckReadyRetries)
	}

	if l.CheckReadyURIPath != r.CheckReadyURIPath {
		diff("CheckReadyURIPath; this %q, other %q", l.CheckReadyURIPath, r.CheckReadyURIPath)
	}

	if l.CheckReadyURITimeout != r.CheckReadyURITimeout {
		diff("CheckReadyURITimeout; this %d, other %d", l.CheckReadyURITimeout, r.CheckReadyURITimeout)
	}

	return diffs
}
