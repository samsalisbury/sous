package cli

import (
	"flag"
	"os"
	"os/exec"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/tempfile"
	"github.com/pkg/errors"
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
	file, err := tempfile.New("", "sous_manifest", ".yaml")
	if err != nil {
		return EnsureErrorResult(err)
	}

	get, err := sme.SousGraph.GetManifestGet(sme.DeployFilterFlags, file, &up)
	if err != nil {
		return EnsureErrorResult(err)
	}

	if err := get.Do(); err != nil {
		return EnsureErrorResult(errors.Wrapf(err, "getting manifest into %s", file.Name()))
	}

	if err := file.Close(); err != nil {
		return EnsureErrorResult(errors.Wrapf(err, "closing file %s", file.Name()))
	}

	if err := doEdit(file.Name()); err != nil {
		return EnsureErrorResult(errors.Wrapf(err, "editing file at %s", file.Name()))
	}

	file, err = os.Open(file.Name())
	if err != nil {
		return EnsureErrorResult(errors.Wrapf(err, "reopening file %s", file.Name()))
	}

	set, err := sme.SousGraph.GetManifestSet(sme.DeployFilterFlags, &up, file)
	if err != nil {
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
	editor.Stdin = os.Stdin
	editor.Stdout = os.Stdout

	return editor.Run()
}
