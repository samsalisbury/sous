package cli

// SourceFlags are CLI flags used to set the source context for a given command
// invocation.
type SourceFlags struct {
	Repo     string
	Offset   string
	Tag      string
	Revision string
}

const sourceFlagsHelp = `
	-repo REPOSITORY_NAME
		source code repository location

		The repository context is the name of a source code repository whose
		code, configuration, artifacts, deployments, etc. will be acted upon.
		If sous is run from inside a Git repository, then repo will default to
		the normalised git-configured fetch URL of any remote named "upstream"
		or "origin", in that order.

		Sous uses go-style repository URLs, and currently only supports GitHub-
		based repositories, e.g. "github.com/user/repo"

	
	-offset RELATIVE_PATH
		source code relative repository offset

		Repository context offset is the relative path within a repository where
		a piece of software is defined.
		
		If you are working in a subdirectory of a repository, the default value
		for offset will be the relative path of the current working directory
		from the repository root.

		Note: if you supply the -repo flag but not -offset, then -offset
		defaults to "".
	
	-tag TAG_NAME
		source code revision tag

		Repository tag is the name of a tag in the repository to act upon.

	-revision REVISION_ID
		source code revision ID

		Revision ID is the ID of a revision in the repository to act upon.
`
