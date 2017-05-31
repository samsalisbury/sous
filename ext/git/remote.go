package git

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

type (
	// Remote represents a Git remote.
	Remote struct {
		Name, PushURL, FetchURL string
	}
	// Remotes is a map of Git remote name to Remote.
	Remotes map[string]Remote
)

// AddFetch adds a fetch URL to the named remote, creating it if it does not
// exist.
func (rs Remotes) AddFetch(name, url string) {
	r := rs.ensureExists(name)
	r.FetchURL = url
	rs[name] = r
}

// AddPush adds a push URL to the named remote, creating it if it does not
// exist.
func (rs Remotes) AddPush(name, url string) {
	r := rs.ensureExists(name)
	r.PushURL = url
	rs[name] = r
}

func (rs Remotes) ensureExists(name string) Remote {
	if _, ok := rs[name]; !ok {
		rs[name] = Remote{}
	}
	return rs[name]
}

// CanonicalRepoURL returns a canonicalised Git repository URL.
// It accepts input URLs using the protocols ssh, git, https, possibly including
// credentials, and possibly ending with ".git", and returns a clean path of the
// form: <hostname>/<repo-path>.
func CanonicalRepoURL(repoURL string) (string, error) {
	if strings.HasPrefix(repoURL, "git@") {
		repoURL = "ssh://" + repoURL
	}
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("only valid URLs can be canonicalised: %s", err)
	}
	host := u.Host
	dir := u.Path
	if host == "" {
		if !strings.ContainsRune(dir, ':') {
			p := strings.SplitN(dir, "/", 2)
			if len(p) != 2 {
				return "", fmt.Errorf("unable to parse %q", repoURL)
			}
			host = p[0]
		}
	}
	if host == "" {
		if !strings.ContainsRune(dir, ':') {
			return "", fmt.Errorf("unable to parse %q", repoURL)
		}
		p := strings.SplitN(dir, ":", 2)
		host = p[0]
		dir = p[1]
	}
	if strings.ContainsRune(host, ':') {
		p := strings.SplitN(host, ":", 2)
		host = p[0]
		dir = p[1] + dir
	}
	if strings.ContainsRune(host, '@') {
		p := strings.SplitN(host, "@", 2)
		host = p[1]
	}
	c := path.Join(host, strings.TrimSuffix(dir, ".git"))
	return c, nil
}
