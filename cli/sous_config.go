package cli

import (
	"github.com/opentable/sous/util/cmdr"
)

// SousConfig is the sous config command.
type SousConfig struct {
	User   LocalUser
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

// Help returns help for 'sous config'.
func (sc *SousConfig) Help() string { return sousConfigHelp }

// Execute displays or sets config properties.
func (sc *SousConfig) Execute(args []string) cmdr.Result {
	switch len(args) {
	default:
		return UsageErrorf("expected 0-2 arguments, received %d", len(args))
	case 0:
		return Successf(sc.Config.String())
	case 1:
		name := args[0]
		v, err := sc.Config.getValue(name)
		if err != nil {
			return UsageErrorf("%s", err)
		}
		return Successf(v)
	case 2:
		name, value := args[0], args[1]
		if err := sc.Config.setValue(sc.User.User, name, value); err != nil {
			return EnsureErrorResult(err)
		}
		return Successf("set %s to %q", name, value)
	}
}
