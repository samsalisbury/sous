package actions

import (
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {

	ls, _ := logging.NewLogSinkSpy()

	p := &PlumbNormalizeGDM{
		Log:           ls,
		StateLocation: "/bogus",
		User:          sous.User{},
	}

	err := p.Do()

	require.Error(t, err)

}
