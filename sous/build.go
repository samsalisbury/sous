package sous

import (
	"fmt"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/shell"
)

type (
	// Build represents a single build of a project.
	Build struct {
		Context                   *SourceContext
		SourceShell, ScratchShell *shell.Sh
		Pack                      Buildpack
	}
	// BuildTarget represents a single target within a Build.
	BuildTarget interface {
		BuildImage()
		BuildContainer()
	}
)

// NewBuild creates a new build using scratchDir as its temporary directory.
// You should ensure that scratchDir is empty.
//func NewBuild(c *SourceContext, scratchDir string) (*Build, error) {
//	sourceShell, err := shell.DefaultInDir(c.AbsDir())
//	if err != nil {
//		return nil, err
//	}
//	scratchShell, err := shell.DefaultInDir(scratchDir)
//	if err != nil {
//		return nil, err
//	}
//	return NewBuildWithShells(c, sourceShell, scratchShell)
//}

// NewBuildWithShells creates a new build using source code in the working
// directory of sourceShell, and using the working dir of scratchShell as
// temporary storage.
func NewBuildWithShells(bp Buildpack, c *SourceContext, sourceShell, scratchShell *shell.Sh) (*Build, error) {
	b := &Build{
		Context:      c,
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

// Start begins the build.
func (b *Build) Start() (*BuildResult, error) {
	bc := &BuildContext{
		Sh: b.SourceShell,
	}
	return b.Pack.Build(bc)
}

// Run does a complete build run
func (b *Build) Run(st *State, ctx *SourceContext, source, scratch *shell.Sh) (*sous.BuildResult, error) {
	bp, err := b.FindBuildpack(st, source)
	if err != nil {
		return nil, err
	}

	build, err := sous.NewBuildWithShells(bp, ctx, source, scratch)
	if err != nil {
		return nil, err
	}

	return build.Start()
}

// FindBuildpack finds the appropriate buildpack for a project
// ( except right now, we just return a hardcoded Dockerfile BP )
// returns an error if no buildpack applies
func FindBuildpack(st *State, s *shell.Sh) (Buildpack, error) {
	return docker.NewDockerfileBuildpack(st.Defs.DockerRepo), nil
}
