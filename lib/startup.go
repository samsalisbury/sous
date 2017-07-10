package sous

import (
	"fmt"
	"sort"
)

// Startup is the configuration for startup checks for a service deployment.
// c.f. DeployConfig for use.
type Startup struct { //                             Singularity fields
	SkipConnectTest bool `yaml:",omitempty"`

	ConnectDelay    int `yaml:",omitempty"` // Healthcheck.StartupDelaySeconds
	Timeout         int `yaml:",omitempty"` // Healthcheck.StartupTimeoutSeconds
	ConnectInterval int `yaml:",omitempty"` // Healthcheck.StartupIntervalSeconds

	SkipReadyTest bool `yaml:",omitempty"`

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

// MergeDefaults merges default values with a Startup and returns the result
func (s Startup) MergeDefaults(base Startup) Startup {
	n := base

	if n.SkipConnectTest == zeroStartup.SkipConnectTest {
		n.SkipConnectTest = s.SkipConnectTest
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

	if n.SkipReadyTest == zeroStartup.SkipReadyTest {
		n.SkipReadyTest = s.SkipReadyTest
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

	if base.SkipConnectTest == s.SkipConnectTest &&
		old.SkipConnectTest == zeroStartup.SkipConnectTest {
		n.SkipConnectTest = zeroStartup.SkipConnectTest
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

	if base.SkipReadyTest == s.SkipReadyTest &&
		old.SkipReadyTest == zeroStartup.SkipReadyTest {
		n.SkipReadyTest = zeroStartup.SkipReadyTest
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
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }

	if s.SkipReadyTest != o.SkipReadyTest {
		diff("SkipReadyTest; this %t, other %t", s.SkipReadyTest, o.SkipReadyTest)
	}

	if s.SkipConnectTest != o.SkipConnectTest {
		diff("SkipConnectTest; this %t, other %t", s.SkipConnectTest, o.SkipConnectTest)
	}

	if s.ConnectDelay != o.ConnectDelay {
		diff("ConnectDelay; this %d, other %d", s.ConnectDelay, o.ConnectDelay)
	}

	if s.ConnectInterval != o.ConnectInterval {
		diff("ConnectInterval; this %d, other %d", s.ConnectInterval, o.ConnectInterval)
	}

	if s.Timeout != o.Timeout {
		diff("Timeout; this %d, other %d", s.Timeout, o.Timeout)
	}

	if s.CheckReadyProtocol != o.CheckReadyProtocol {
		diff("CheckReadyProtocol; this %q, other %q", s.CheckReadyProtocol, o.CheckReadyProtocol)
	}

	if s.CheckReadyPortIndex != o.CheckReadyPortIndex {
		diff("CheckReadyPortIndex; this %d, other %d", s.CheckReadyPortIndex, o.CheckReadyPortIndex)
	}

	if len(s.CheckReadyFailureStatuses) != len(o.CheckReadyFailureStatuses) {
		diff("CheckReadyFailureStatuses; this %v, other %v", s.CheckReadyFailureStatuses, o.CheckReadyFailureStatuses)
	} else {
		sort.Ints(o.CheckReadyFailureStatuses)
		sort.Ints(s.CheckReadyFailureStatuses)
		for n, sfs := range s.CheckReadyFailureStatuses {
			if sfs != o.CheckReadyFailureStatuses[n] {
				diff("CheckReadyFailureStatuses; this %v, other %v", s.CheckReadyFailureStatuses, o.CheckReadyFailureStatuses)
				break
			}
		}
	}

	if s.CheckReadyInterval != o.CheckReadyInterval {
		diff("CheckReadyInterval; this %d, other %d", s.CheckReadyInterval, o.CheckReadyInterval)
	}

	if s.CheckReadyRetries != o.CheckReadyRetries {
		diff("CheckReadyRetries; this %d, other %d", s.CheckReadyRetries, o.CheckReadyRetries)
	}

	if s.CheckReadyURIPath != o.CheckReadyURIPath {
		diff("CheckReadyURIPath; this %q, other %q", s.CheckReadyURIPath, o.CheckReadyURIPath)
	}

	if s.CheckReadyURITimeout != o.CheckReadyURITimeout {
		diff("CheckReadyURITimeout; this %d, other %d", s.CheckReadyURITimeout, o.CheckReadyURITimeout)
	}

	return diffs
}
