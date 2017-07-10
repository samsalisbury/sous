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
		Startup{false, "/health", 100, 10},
		Startup{}, //zero base
	)

	s.PutGet(
		Startup{}, //zero default
		Startup{false, "/health", 100, 10},
		Startup{false, "/health", 0, 0},
	)

	s.PutGet(
		Startup{false, "/health", 100, 0},
		Startup{false, "/health", 100, 10},
		Startup{true, "", 0, 0},
	)
}

func (s *StartupTest) GetPut(defaults, base Startup) {
	s.Equal(base, defaults.UnmergeDefaults(defaults.MergeDefaults(base), base))
}

func (s *StartupTest) TestGetPut() {
	s.GetPut(
		Startup{}, // zero default
		Startup{false, "/health", 100, 10},
	)

	s.GetPut(
		Startup{}, // zero default
		Startup{false, "/health", 100, 10},
	)

	s.GetPut(
		Startup{false, "/heath", 100, 0},
		Startup{true, "", 0, 10},
	)
}

// PutPut is guaranteed by the instance receiver
