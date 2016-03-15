package main

import (
	"fmt"
	"os"
)

type VersionCommand struct {
	Version SousVersion
}

const versionCommandHelp = `
version prints the current version and revision of sous
`

func (v *VersionCommand) Help() string {
	return versionCommandHelp
}

func (vc *VersionCommand) Execute() error {
	_, err := fmt.Fprintln(os.Stdout, vc.Version)
	return err
}
