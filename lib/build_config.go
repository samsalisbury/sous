package sous

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

type (
	// BuildConfig captures the user's intent as they build a repo.
	BuildConfig struct {
		Repo, Offset, Tag, Revision string
		Strict, ForceClone, Dev     bool
		Context                     *BuildContext
		LogSink                     logging.LogSink
	}
)

// NewContext returns a new BuildContext updated based on the user's intent as expressed in the Config
func (c *BuildConfig) NewContext() *BuildContext {
	ctx := c.Context
	sc := c.Context.Source
	sh := ctx.Sh.Clone()
	tag := c.chooseTag()
	sh.CD(sc.RootDir)
	bc := BuildContext{
		Sh:      sh,
		Scratch: ctx.Scratch,
		Machine: ctx.Machine,
		User:    ctx.User,
		Changes: ctx.Changes,
		Source: SourceContext{
			OffsetDir:      c.chooseOffset(),
			RemoteURL:      c.chooseRemoteURL(),
			NearestTagName: tag,

			RootDir:            sc.RootDir,
			Branch:             sc.Branch,
			Revision:           sc.Revision,
			Files:              sc.Files,
			ModifiedFiles:      sc.ModifiedFiles,
			NewFiles:           sc.NewFiles,
			Tags:               sc.Tags,
			NearestTagRevision: sc.NearestTagRevision,
			NearestTag:         Tag{Name: tag, Revision: sc.NearestTagRevision},
			PrimaryRemoteURL:   sc.PrimaryRemoteURL,
			RemoteURLs:         sc.RemoteURLs,
			DirtyWorkingTree:   sc.DirtyWorkingTree,
			RevisionUnpushed:   sc.RevisionUnpushed,
			DevBuild:           c.Dev,
		},
	}

	bc.Advisories = c.Advisories(&bc)

	return &bc
}

func (c *BuildConfig) chooseRemoteURL() string {
	if c.Repo == "" {
		messages.ReportLogFieldsMessage("Using best guest", logging.DebugLevel, c.LogSink, c.Context.Source.PrimaryRemoteURL)
		return c.Context.Source.PrimaryRemoteURL
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

// Resolve settles configurations so that e.g. captured version tags are used in the absence of user input
func (c *BuildConfig) Resolve() {
	c.Tag = c.chooseTag()
}

// Validate checks that the Config is well formed
func (c *BuildConfig) Validate() error {
	_, _, err := parseSemverTagWithOptionalPrefix(c.Tag)
	return err
}

// GuardStrict returns an error if there are imperfections in the proposed build
func (c *BuildConfig) GuardStrict(bc *BuildContext) error {
	if !c.Strict {
		return nil
	}
	as := bc.Advisories
	if len(as) > 0 {
		return fmt.Errorf("Strict built encountered advisories:\n  %s", strings.Join(as.Strings(), "  \n"))
	}
	return nil
}

// GuardRegister returns an error if any development-only advisories exist
func (c *BuildConfig) GuardRegister(br *BuildResult) error {
	var blockers []string
	for _, p := range br.Products {
		for _, a := range p.Advisories {
			switch AdvisoryName(a) {
			case DirtyWS, UnpushedRev, NoRepoAdv, NotRequestedRevision:
				blockers = append(blockers, fmt.Sprintf("%s: %s", p.Source.String(), a))
			}
		}
	}
	if len(blockers) > 0 {
		return fmt.Errorf("build may not be deployable in all clusters due to advisories:\n  %s", strings.Join(blockers, "\n  "))
	}
	return nil
}

// Advisories returns a list of advisories that apply to ctx.
func (c *BuildConfig) Advisories(ctx *BuildContext) Advisories {
	advs := Advisories{}
	s := ctx.Source
	knowsRepo := false
	for _, r := range s.RemoteURLs {
		if s.RemoteURL == r {
			knowsRepo = true
			break
		}
	}
	if !knowsRepo {
		advs = append(advs, UnknownRepo)
	}

	if s.RemoteURL == "" {
		advs = append(advs, NoRepoAdv)
	}

	if c.Revision != "" && c.Revision != s.Revision {
		advs = append(advs, NotRequestedRevision)
	}

	if c.Context.Source.Version().Version.Format(`M.m.p`) == `0.0.0` {
		advs = append(advs, Unversioned)
	}

	if c.Tag != "" {
		hasTag := false
		for _, t := range s.Tags {
			if t.Name == c.Tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			advs = append(advs, EphemeralTag)
		} else if s.NearestTagRevision != s.Revision {
			messages.ReportLogFieldsMessage("NearestTagRevision != Revision", logging.DebugLevel, c.LogSink, s.NearestTagRevision, s.Revision)
			advs = append(advs, TagNotHead)
		}
	}

	if s.DirtyWorkingTree {
		advs = append(advs, DirtyWS)
	}

	if s.RevisionUnpushed {
		advs = append(advs, UnpushedRev)
	}

	if c.Dev {
		advs = append(advs, DeveloperBuild)
	}

	return advs
}
