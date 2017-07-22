package sous

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StartupTest struct {
	suite.Suite
}

// Inspired by:
// http://sebfisch.github.io/research/pub/Fischer+MPC15.pdf
// I assert that get:MergeDefaults and putback:UnmergeDefaults comprise well behaved lens.

func TestStartup(t *testing.T) {
	suite.Run(t, new(StartupTest))
}

/*
func (s *StartupTest) SetupTest() {
}
*/

func (s *StartupTest) PutGet(defaults, merged, base Startup) {
	s.Equal(merged, defaults.MergeDefaults(defaults.UnmergeDefaults(merged, base)))
}

func (s *StartupTest) TestPutGet() {
	s.PutGet(
		Startup{}, //zero default
		Startup{CheckReadyURIPath: "/health", CheckReadyURITimeout: 100, Timeout: 10},
		Startup{}, //zero base
	)

	s.PutGet(
		Startup{CheckReadyFailureStatuses: []int{410, 503}},
		Startup{CheckReadyFailureStatuses: []int{410, 503}},
		Startup{}, //zero base
	)

	s.PutGet(
		Startup{}, //zero default
		Startup{CheckReadyFailureStatuses: []int{410, 503}},
		Startup{CheckReadyFailureStatuses: []int{410, 503}},
	)

	s.PutGet(
		Startup{}, //zero default
		Startup{CheckReadyURIPath: "/health", CheckReadyURITimeout: 100, Timeout: 10},
		Startup{CheckReadyURIPath: "/health"},
	)

	s.PutGet(
		Startup{CheckReadyURIPath: "/health", CheckReadyURITimeout: 100, Timeout: 0},
		Startup{CheckReadyURIPath: "/health", CheckReadyURITimeout: 100, Timeout: 10},
		Startup{SkipCheck: true},
	)
}

func (s *StartupTest) GetPut(defaults, base Startup) {
	s.Equal(base, defaults.UnmergeDefaults(defaults.MergeDefaults(base), base))
}

func (s *StartupTest) TestGetPut() {
	s.GetPut(
		Startup{}, // zero default
		Startup{CheckReadyURIPath: "/health", CheckReadyURITimeout: 100, Timeout: 10},
	)

	s.GetPut(
		Startup{}, // zero default
		Startup{CheckReadyURIPath: "/health", CheckReadyURITimeout: 100, Timeout: 10},
	)

	s.GetPut(
		Startup{CheckReadyURIPath: "/heath", CheckReadyURITimeout: 100, Timeout: 0},
		Startup{SkipCheck: true, Timeout: 10},
	)

	s.GetPut(
		Startup{}, // zero default
		Startup{CheckReadyFailureStatuses: []int{410, 503}},
	)

	s.GetPut(
		Startup{
			ConnectDelay:    234,
			ConnectInterval: 678,
			SkipCheck:       true,
		},
		Startup{
			CheckReadyProtocol:  "https",
			CheckReadyPortIndex: 2,
			CheckReadyInterval:  978,
			CheckReadyRetries:   67,
		},
	)

}

func (s *StartupTest) TestMerge() {
	left := Startup{
		ConnectDelay:         234,
		ConnectInterval:      678,
		SkipCheck:            true,
		CheckReadyURITimeout: 100,
		Timeout:              10,
	}
	right := Startup{
		CheckReadyURIPath:         "/health",
		CheckReadyFailureStatuses: []int{410, 503},
		CheckReadyProtocol:        "https",
		CheckReadyPortIndex:       2,
		CheckReadyInterval:        978,
		CheckReadyRetries:         67,
	}

	merged := Startup{
		CheckReadyURIPath:         "/health",
		CheckReadyURITimeout:      100,
		Timeout:                   10,
		CheckReadyFailureStatuses: []int{410, 503},
		ConnectDelay:              234,
		ConnectInterval:           678,
		SkipCheck:                 true,
		CheckReadyProtocol:        "https",
		CheckReadyPortIndex:       2,
		CheckReadyInterval:        978,
		CheckReadyRetries:         67,
	}

	s.Equal(merged, left.MergeDefaults(right))
	s.Equal(merged, right.MergeDefaults(left))

}

// PutPut is guaranteed by the instance receiver
