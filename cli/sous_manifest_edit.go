package cli

import (
	"flag"
	"os"
	"os/exec"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousManifestGet defines the `sous manifest edit` command.
type SousManifestEdit struct {
	config.DeployFilterFlags `inject:"optional"`
	SousGraph                *graph.SousGraph
}

func init() { ManifestSubcommands["edit"] = &SousManifestEdit{} }

const sousManifestGetHelp = `edit a deployment manifest`

// Help implements Command on SousManifestEdit.
func (*SousManifestEdit) Help() string { return sousManifestHelp }

// AddFlags implements AddFlagger on SousManifestEdit.
func (sme *SousManifestEdit) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sme.DeployFilterFlags, ManifestFilterFlagsHelp)
}

// Execute implements Executor on SousManifestEdit.
func (sme *SousManifestEdit) Execute(args []string) cmdr.Result {
	get, err := sme.SousGraph.GetManifestGet(sme.DeployFilterFlags)
	if err != nil {
		return EnsureErrorResult(err)
	}
	set, err := sme.SousGraph.GetManifestSet(sme.DeployFilterFlags)
	if err != nil {
		return EnsureErrorResult(err)
	}

	if err := get.Do(); err != nil {
		return EnsureErrorResult(err)
	}

	if err := doEdit(); err != nil {
		return EnsureErrorResult(err)
	}

	if err := set.Do(); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}

func doEdit(path string) error {
	// edit goes here
	editCmd, set := os.LookupEnv("EDITOR")
	if !set {
		editCmd = "vi"
	}
	editor := exec.Command(editCmd, path)
	return editor.Run()
}
