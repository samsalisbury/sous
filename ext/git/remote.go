package git

import (
	"fmt"
	"net/url"
	"strings"
)

type (
	Remote struct {
		Name, PushURL, FetchURL string
	}
	Remotes map[string]Remote
)

func (rs Remotes) AddFetch(name, url string) {
	r := rs.ensureExists(name)
	r.FetchURL = url
	rs[name] = r
}

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

func CanonicalRepoURL(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("only valid URLs can be canonicalised: %s", err)
	}
	host := u.Host
	path := u.Path
	if host == "" {
		if !strings.ContainsRune(path, ':') {
			return "", fmt.Errorf("URL %q contains neither host nor username", u)
		}
		p := strings.SplitN(path, ":", 2)
		host = p[0]
		path = "/" + p[1]
	}
	if strings.ContainsRune(host, '@') {
		p := strings.SplitN(host, "@", 2)
		host = p[1]
	}
	return host + strings.TrimSuffix(path, ".git"), nil
}
