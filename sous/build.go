package sous

import (
	"fmt"

	"github.com/opentable/sous/util/shell"
)

type (
	Build struct {
		Source                    SourceContext
		SourceShell, ScratchShell *shell.Sh
	}
	BuildTarget interface {
		BuildImage()
		BuildContainer()
	}
)

// NewBuild creates a new build using source code at sourceDir, and using
// scratchDir as its temporary directory. You should ensure that scratchDir is
// empty.
func NewBuild(sourceDir, scratchDir string) (*Build, error) {
	sourceShell, err := shell.DefaultInDir(sourceDir)
	if err != nil {
		return nil, err
	}
	scratchShell, err := shell.DefaultInDir(scratchDir)
	if err != nil {
		return nil, err
	}
	return NewBuildWithShells(sourceShell, scratchShell)
}

// NewBuildWithShells creates a new build using source code in the working
// directory of sourceShell, and using the working dir of scratchShell as
// temporary storage.
func NewBuildWithShells(sourceShell, scratchShell *shell.Sh) (*Build, error) {
	b := &Build{
		SourceShell:  sourceShell,
		ScratchShell: scratchShell,
	}
	files, err := scratchShell.List()
	if err != nil {
		return nil, err
	}
	if len(files) != 0 {
		return nil, fmt.Errorf("scratch dir %s was not empty", scratchShell.Dir)
	}
	return b, nil
}

func (b *Build) Start() error {
	b.createCompileImage()
	panic("not implemented")
}

func (b *Build) createCompileImage() {
	// In the scratch dir:
	//   1. Copy all source files here.
	//   2. Make a new dir, add the scripts.
	//   3. Write the Dockerfile to the scratch dir
}
