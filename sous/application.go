package sous

type (
	Applications []Application
	Application  struct {
		Source Source
	}
	Source struct {
		RepoURL RepoURL
		RepoDir string
	}
	RepoURL string
)
