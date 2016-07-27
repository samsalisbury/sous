package sous

import "path/filepath"

type (
	// BuildConfig captures the user's intent as they build a repo
	BuildConfig struct {
		Repo, Offset, Tag, Revision string
		Strict, ForceClone          bool
		Context                     *BuildContext
	}

	// An AdvisoryName is the type for advisory tokens
	AdvisoryName string
	Advisories   []AdvisoryName
)

const (
	UnknownRepo          = AdvisoryName(`source workspace lacked repo`)
	NoRepoAdv            = AdvisoryName(`no repository`)
	NotRequestedRevision = AdvisoryName(`requested revision not built`)
	Unversioned          = AdvisoryName(`no versioned tag`)
	TagMismatch          = AdvisoryName(`tag mismatch`)
	TagNotHead           = AdvisoryName(`tag not on built revision`)
	EphemeralTag         = AdvisoryName(`ephemeral tag`)
	UnpushedRev          = AdvisoryName(`unpushed revision`)
	BogusRev             = AdvisoryName(`bogus revision`)
	DirtyWS              = AdvisoryName(`dirty workspace`)
)

// NewContext returns a new BuildContext updated based on the user's intent as expressed in the Config
func (c *BuildConfig) NewContext() *BuildContext {
	ctx := c.Context
	sc := c.Context.Source
	return &BuildContext{
		Sh:         ctx.Sh,
		Scratch:    ctx.Scratch,
		Machine:    ctx.Machine,
		User:       ctx.User,
		Changes:    ctx.Changes,
		Advisories: c.Advisories(),
		Source: SourceContext{
			RootDir:                  sc.RootDir,
			OffsetDir:                c.chooseOffset(),
			Branch:                   sc.Branch,
			Revision:                 sc.Revision,
			Files:                    sc.Files,
			ModifiedFiles:            sc.ModifiedFiles,
			NewFiles:                 sc.NewFiles,
			Tags:                     sc.Tags,
			NearestTagName:           c.chooseTag(),
			NearestTagRevision:       sc.NearestTagRevision,
			RemoteURL:                c.chooseRemoteURL(),
			PossiblePrimaryRemoteURL: sc.PossiblePrimaryRemoteURL,
			RemoteURLs:               sc.RemoteURLs,
			DirtyWorkingTree:         sc.DirtyWorkingTree,
		},
	}
}

func (c *BuildConfig) chooseRemoteURL() string {
	if c.Repo == "" {
		return c.Context.Source.PossiblePrimaryRemoteURL
	}
	return c.Repo
}

func (c *BuildConfig) chooseTag() string {
	if c.Tag == "" {
		return c.Context.Source.NearestTagName
	}
	return c.Tag
}

func (c *BuildConfig) chooseOffset() string {
	if c.Offset == "" {
		return c.Context.Source.OffsetDir
	}
	clean := filepath.Clean(c.Offset)
	if clean == "." {
		return ""
	}
	return clean
}

// Advisories does this:
func (c *BuildConfig) Advisories() (advs []string) {
	s := c.Context.Source
	knowsRepo := false
	for _, r := range s.RemoteURLs {
		if s.RemoteURL == r {
			knowsRepo = true
			break
		}
	}
	if !knowsRepo {
		advs = append(advs, string(UnknownRepo))
	}

	if c.Repo == "" && s.RemoteURL == "" {
		advs = append(advs, string(NoRepoAdv))
	}

	if c.Revision != s.Revision {
		advs = append(advs, string(NotRequestedRevision))
	}

	if c.Context.Source.Version().Version.Format(`M.m.p`) == `0.0.0` {
		advs = append(advs, string(Unversioned))
	}

	if s.NearestTagRevision != s.Revision {
		advs = append(advs, string(TagNotHead))
	}

	hasTag := false
	for _, t := range s.Tags {
		if t.Name == c.Tag {
			hasTag = true
			break
		}
	}
	if !hasTag {
		advs = append(advs, string(EphemeralTag))
	}

	if s.DirtyWorkingTree {
		advs = append(advs, string(DirtyWS))
	}

	if s.RevisionUnpushed {
		advs = append(advs, string(UnpushedRev))
	}

	/*
		BuildContext struct {
			Sh      shell.Shell
			Source  SourceContext
			Scratch ScratchContext
			Machine Machine
			User    user.User
			Changes Changes
		}

		SourceContext struct {
			RootDir, OffsetDir, Branch, Revision string
			Files, ModifiedFiles, NewFiles       []string
			Tags                                 []Tag
			NearestTagName, NearestTagRevision   string
			PossiblePrimaryRemoteURL             string
			DirtyWorkingTree                     bool
		}
	*/
	return
}
