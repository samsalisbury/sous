package config

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestLoggingConfig(t *testing.T) {
	testcase := func(silent, quiet, loud, debug bool, lvl logging.Level) {
		t.Run(fmt.Sprintf("%t %t %t %t -> %s", silent, quiet, loud, debug, lvl), func(t *testing.T) {
			v := Verbosity{Silent: silent, Quiet: quiet, Loud: loud, Debug: debug}
			cfg := v.LoggingConfiguration()
			assert.NoError(t, cfg.Validate())

			assert.Equal(t, lvl.String(), cfg.GetBasicLevel().String())
			assert.Equal(t, lvl, cfg.GetBasicLevel())
		})
	}

	testcase(false, false, false, false, logging.WarningLevel)
	testcase(true, false, false, false, logging.CriticalLevel)
	testcase(false, true, false, false, logging.CriticalLevel)
	testcase(false, false, true, false, logging.ExtraDebug1Level)
	testcase(false, false, false, true, logging.DebugLevel)
	testcase(true, false, false, true, logging.CriticalLevel)
}
