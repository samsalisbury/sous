package cli

import (
	"github.com/opentable/sous/util/cmdr"
)

type SousConfig struct {
	User   *User
	Config LocalSousConfig
}

func init() { TopLevelCommands["config"] = &SousConfig{} }

const sousConfigHelp = `
view and edit sous configuration

usage: sous config [<key> [value]]

Invoking sous config with no arguments lists all configuration key/value pairs.
If you pass just a single argument (a key) sous config will output just the
value of that key. You can set a key by providing both a key and a value.
`

func (sc *SousConfig) Help() string { return sousConfigHelp }

func (sc *SousConfig) Execute(args []string) cmdr.Result {
	switch len(args) {
	default:
		return UsageErrorf("expected 0-2 arguments, received %d", len(args))
	case 0:
		return Successf(sc.Config.String())
	case 1:
		v, err := sc.Config.GetValue(args[0])
		if err != nil {
			return UsageErrorf("%s", err)
		}
		return Successf(v)
	case 2:
		return InternalErrorf("setting config values not implemented")
	}
}
