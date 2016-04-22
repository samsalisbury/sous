package git

import (
	"path/filepath"

	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/parallel"
)

// SourceContext gathers together a number of bits of information about the
// repository such as its current branch, revision, nearest tag, nearest semver
// tag, etc.
func (r *Repo) SourceContext() (*sous.SourceContext, error) {
	var (
		revision, branch, nearestTagName,
		repoRelativeDir string
		files, modifiedFiles, newFiles []string
		allTags                        []sous.Tag
		remotes                        Remotes
	)
	c := r.Client
	if err := parallel.Do(
		func(err *error) { branch, *err = c.CurrentBranch() },
		func(err *error) { revision, *err = c.Revision() },
		func(err *error) {
			repoRelativeDir, *err = filepath.Rel(r.Root, c.Sh.Dir)
			if repoRelativeDir == "." {
				repoRelativeDir = ""
			}
		},
		func(err *error) {
			allTags, *err = r.Client.ListTags()
			if err != nil || len(allTags) == 0 {
				return
			}
			nearestTagName, *err = c.NearestTag()
			if err != nil {
				return
			}
			//nearestTagRevision, *err = c.RevisionAt(nearestTagName)
		},
		func(err *error) { files, *err = c.ListFiles() },
		func(err *error) { modifiedFiles, *err = c.ModifiedFiles() },
		func(err *error) { newFiles, *err = c.NewFiles() },
		func(err *error) { remotes, *err = c.ListRemotes() },
	); err != nil {
		return nil, err
	}

	primaryRemoteURL := guessPrimaryRemote(remotes)

	return &sous.SourceContext{
		RootDir:                  r.Root,
		OffsetDir:                repoRelativeDir,
		Branch:                   branch,
		Revision:                 revision,
		Files:                    files,
		ModifiedFiles:            modifiedFiles,
		NewFiles:                 newFiles,
		Tags:                     allTags,
		NearestTagName:           nearestTagName,
		PossiblePrimaryRemoteURL: primaryRemoteURL,
		DirtyWorkingTree:         len(modifiedFiles)+len(newFiles) != 0,
	}, nil
}

func guessPrimaryRemote(remotes map[string]Remote) string {
	primaryRemote, ok := remotes["upstream"]
	if !ok {
		primaryRemote, ok = remotes["origin"]
	}
	if !ok {
		return ""
	}
	if primaryRemote.FetchURL == "" {
		return ""
	}
	// We don't care about this error, empty string is
	// an acceptable return value.
	u, _ := CanonicalRepoURL(primaryRemote.FetchURL)
	return u
}
