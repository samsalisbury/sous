package sous

type (
	// BuildConfig captures the user's intent as they build a repo
	BuildConfig struct {
		Repo, Offset, Tag, Revision string
		Strict, ForceClone          bool
		Context                     *BuildContext
	}

	// An AdvisoryName is the type for advisory tokens
	AdvisoryName string
)

const (
	UnknownRepo  = AdvisoryName(`source workspace lacked repo`)
	NoRepoAdv    = AdvisoryName(`no repository`)
	TagMismatch  = AdvisoryName(`tag mismatch`)
	EphemeralTag = AdvisoryName(`ephemeral tag`)
	UnpushedRev  = AdvisoryName(`unpushed revision`)
	BogusRev     = AdvisoryName(`bogus revision`)
	DirtyWS      = AdvisoryName(`dirty workspace`)
)

// NewContext returns a new BuildContext updated based on the user's intent as expressed in the Config
func (c *BuildConfig) NewContext() *BuildContext {
	ctx := c.Context
	sc := c.Context.Source
	return &BuildContext{
		Sh:      ctx.Sh,
		Scratch: ctx.Scratch,
		Machine: ctx.Machine,
		User:    ctx.User,
		Changes: ctx.Changes,
		Source: SourceContext{
			RootDir:                  sc.RootDir,
			OffsetDir:                sc.OffsetDir,
			Branch:                   sc.Branch,
			Revision:                 sc.Revision,
			Files:                    sc.Files,
			ModifiedFiles:            sc.ModifiedFiles,
			NewFiles:                 sc.NewFiles,
			Tags:                     sc.Tags,
			NearestTagName:           sc.NearestTagName,
			NearestTagRevision:       sc.NearestTagRevision,
			RemoteURL:                c.chooseRemoteURL(),
			PossiblePrimaryRemoteURL: sc.PossiblePrimaryRemoteURL,
			RemoteURLs:               sc.RemoteURLs,
			DirtyWorkingTree:         sc.DirtyWorkingTree,
		},
	}
}

func (c *BuildConfig) chooseRemoteURL() string {
	if Repo == "" {
		return c.Context.Source.PossiblePrimaryRemoteURL
	}
	return Repo
}

// Advisories does this:
func (c *BuildConfig) Advisories() ([]string, error) {
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
	return nil, nil
}
