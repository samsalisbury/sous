package cli

import "github.com/samsalisbury/semv"

// Flags defines the flags available to all sous commands.
type Flags struct {
	Repo       string       `flag:"repo"`
	Offset     string       `flag:"offset"`
	Version    semv.Version `flag:"version"`
	Tag        string       `flag:"tag"`
	Revision   string       `flag:"revision"`
	Strict     bool         `flag:"strict"`
	Deployer   string       `flag:"deployer"`
	Builder    string       `flag:"builder"`
	DryRun     bool         `flag:"dry-run"`
	ForceClone bool         `flag:"force-clone"`
	Cluster    string       `flag:"cluster"`
}

var flagDescriptions = `
	-repo=REPOSITORY_NAME
		set the repository context

		The repository context is the name of a source code repository whose
		code, configuration, artifacts, deployments, etc. will be acted upon.
		If sous is run from inside a Git repository, then repo will default to
		the normalised git-configured fetch URL of any remote named "upstream"
		or "origin", in that order.

		Sous uses go-style repository URLs, and currently only supports GitHub-
		based repositories, e.g. "github.com/user/repo"

	
	-offset=RELATIVE_PATH
		set the repository context offset

		Repository context offset is the relative path within a repository where
		a piece of software is defined.
		
		If you are working in a subdirectory of a repository, the default value
		for offset will be the relative path of the current working directory
		from the repository root.

		Note: if you supply the -repo flag but not -offset, then -offset
		defaults to "".
	
	-
`
