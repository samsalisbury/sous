package sous

type (
	// An AdvisoryName is the type for advisory tokens.
	AdvisoryName string
	// Advisories are the advisory tokens that apply to a build
	Advisories []AdvisoryName
)

// Contains returns true if as contains a.
func (as Advisories) Contains(a AdvisoryName) bool {
	for _, b := range as {
		if b == a {
			return true
		}
	}
	return false
}

const (
	// NotService is an advisory that this container is not a service, but
	// instead a support container of some kind and should not itself be
	// deployed.
	NotService = AdvisoryName(`support container`)
	// IsBuilder is an advisory that this container was used to build a finished
	// image, and should not itself be deployed.
	IsBuilder = AdvisoryName(`is a build image`)
	// UnknownRepo is an advisory that the source workspace is not a repo.
	// TODO: Disambiguate text from NoRepoAdv, they seem like the same thing.
	UnknownRepo = AdvisoryName(`source workspace lacked repo`)
	// NoRepoAdv means there is no repository.
	// TODO: Disambiguate text from UnknownRepo.
	NoRepoAdv = AdvisoryName(`no repository`)
	// NotRequestedRevision means that a different revision was built from that
	// which was requested.
	NotRequestedRevision = AdvisoryName(`requested revision not built`)
	// Unversioned means that there was no tag at the currently checked out
	// revision, or that the tag was not a semver tag, or the tag was 0.0.0.
	Unversioned = AdvisoryName(`no versioned tag`)
	// TagMismatch means that a different tag to the one which was requested was
	// built.
	TagMismatch = AdvisoryName(`tag mismatch`)
	// TagNotHead means that the requested tag exists in the history, but there
	// were more commits since, which were part of this build.
	TagNotHead = AdvisoryName(`tag not on built revision`)
	// EphemeralTag means the tag was an ephemeral tag rather than an annotated
	// tag.
	EphemeralTag = AdvisoryName(`ephemeral tag`)
	// UnpushedRev means the revision that was build is not pushed to any
	// remote.
	UnpushedRev = AdvisoryName(`unpushed revision`)
	// BogusRev means that the revision was bogus.
	// TODO: Find out what "bogus" means.
	BogusRev = AdvisoryName(`bogus revision`)
	// DirtyWS means that the workspace was dirty, which means there were
	// untracked files present, or that one or more tracked files were modified
	// since the last commit.
	DirtyWS = AdvisoryName(`dirty workspace`)
	// DeveloperBuild means image was built with the dev flag true, only enables
	// local image detection at the moment.
	DeveloperBuild = AdvisoryName(`developer build`)
	// AddedArtifact means artifact was added to Sous manually and may not have
	// been built by sous.
	AddedArtifact = AdvisoryName(`added artifact`)
)

// AllAdvisories returns all advisories.
func AllAdvisories() Advisories {
	return Advisories{
		NotService,
		IsBuilder,
		UnknownRepo,
		NoRepoAdv,
		NotRequestedRevision,
		Unversioned,
		TagMismatch,
		TagNotHead,
		EphemeralTag,
		UnpushedRev,
		BogusRev,
		DirtyWS,
		DeveloperBuild,
		AddedArtifact,
	}
}

// AllAdvisoryStrings is similar to AllAdvisories except it casts them to
// strings.
func AllAdvisoryStrings() []string {
	return AllAdvisories().Strings()
}

// Strings returns as as a slice of strings.
func (as Advisories) Strings() []string {
	s := make([]string, len(as))
	for i, a := range as {
		s[i] = string(a)
	}
	return s
}
