package cli

type SousConfig struct {
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
