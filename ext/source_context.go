package ext

import (
	"fmt"

	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/sous"
)

//SourceContext struct {
//	Branch, Revision, OffsetDir  string
//	Files                        []string
//	NearestTag, NearestSemverTag Tag
//	DirtyWorkingTree             bool
//}

type SourceContextProvider struct {
	Git *git.Client
}

func (scp *SourceContextProvider) Get(dir string) (*sous.SourceContext, error) {
	gitRepo, err := scp.Git.OpenRepo(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to provide source context using git: %s",
			err)
	}
	return scp.GitSourceContext(gitRepo)
}

func (scp *SourceContextProvider) GitSourceContext(repo *git.Repo) (*sous.SourceContext, error) {
	c, err := repo.Context()
	if err != nil {
		return nil, err
	}
	return &sous.SourceContext{
		Branch:           "",
		Revision:         c.Revision,
		OffsetDir:        c.RepoRelativeDir,
		Files:            repo.Files(),
		NearestTag:       c.NearestTag,
		DirtyWorkingTree: "",
	}

}
