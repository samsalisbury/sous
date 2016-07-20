package sous

// BuildConfig captures the user's intent as they build a repo
type BuildConfig struct {
	Repo, Offset, Tag, Revision string
	Strict, ForceClone          bool
}

// ComputeAdvisories does this:
// If --repo is present, and we're in a git workspace, compare the --repo to
// the remotes of the workspace. If it's present, assume that we're working in
// the current workspace. If it's absent, we'll be building from a shallow
// clone of the given --repo.
// If --repo is absent, guess the repo from the
// remotes of the current workspace: first the upstream workspace, then the
// origin. If neither are present on the current workspace (or we're not in a
// git workspace), add the advisory "no repo." We're now either working locally
// (in the git workspace) or in a clone.
// If --force-clone is present, we ignore
// the presence of a valid workspace and do a shallow clone anyway.
// Now, capture the tag and revision. First, prefer the appropriate --tag and
// --revision flags. If both are missing and we're working locally, use the
// current workspace HEAD revision. If --tag is missing, use the "closest
// semver" tag to the revision determined (either HEAD or --revision). If only
// --tag is specified, and the named tag exists and points to a revision, use
// that revision. If the tag is missing or doesn't point to a revision, use
// HEAD.
// Check that the tag and the revision align (i.e. that the tag in the repo
// points to the named revision). If the tag exists and points to a different
// revision, add the "tag mismatch" advisory. If the tag doesn't exist in the
// remote, add the "ephemeral tag" advisory. If the revision isn't present in
// the remote repo, add "unpushed revision". If it also isn't present in the
// current workspace, add "bogus revision."
// If --offset is absent, and we're working locally, set offset to the relative
// path from the root of the workspace (or empty in the special case that
// they're identical.)
// If we're working locally, and there are modified files, add the "dirty
// workspace" advisory.
// Issue warnings to the user of any advisories on the build, perform the
// build. --strict behaves like an "errors are warnings" feature, and refuses
// to build if there are advisories.
func ComputeAdvisories(c *BuildConfig, ctx *BuildContext) ([]string, error) {
	return nil
}
