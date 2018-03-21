package cli

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/restful"
)

// SousManifestEdit defines the `sous manifest edit` command.
type SousManifestEdit struct {
	config.DeployFilterFlags `inject:"optional"`
	SousGraph                *graph.SousGraph
}

func init() { ManifestSubcommands["edit"] = &SousManifestEdit{} }

const sousManifestEditHelp = `edit a deployment manifest`

// Help implements Command on SousManifestEdit.
func (*SousManifestEdit) Help() string { return sousManifestEditHelp }

// AddFlags implements AddFlagger on SousManifestEdit.
func (sme *SousManifestEdit) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sme.DeployFilterFlags, ManifestFilterFlagsHelp)
}

// Execute implements Executor on SousManifestEdit.
func (sme *SousManifestEdit) Execute(args []string) cmdr.Result {
	var up restful.Updater
	file, err := ioutil.TempFile("", "sous_manifest")

	if err != nil {
		return EnsureErrorResult(err)
	}

	get, err := sme.SousGraph.GetManifestGet(sme.DeployFilterFlags, file, func(u restful.Updater) {
		up = u
	})
	if err != nil {
		return EnsureErrorResult(err)
	}

	set, err := sme.SousGraph.GetManifestSet(sme.DeployFilterFlags, up, file)
	if err != nil {
		return EnsureErrorResult(err)
	}

	if err := get.Do(); err != nil {
		return EnsureErrorResult(err)
	}

	if err := doEdit(file.Name()); err != nil {
		return EnsureErrorResult(err)
	}

	if _, err := file.Seek(0, 0); err != nil {
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
