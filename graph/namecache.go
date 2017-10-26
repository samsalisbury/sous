package graph

import (
	"sync"

	"github.com/opentable/sous/ext/docker"
	"github.com/pkg/errors"
)

func newLazyNameCache(cfg LocalSousConfig, ls LogSink, cl LocalDockerClient) lazyNameCache {
	return func() (*docker.NameCache, error) {
		theNameCacheOnce.Do(func() {
			theNameCache, theNameCacheErr = generateNameCache(cfg, ls, cl)
		})
		return theNameCache, theNameCacheErr
	}
}

func newNameCache(f lazyNameCache) (*docker.NameCache, error) {
	return f()
}

type lazyNameCache func() (*docker.NameCache, error)

var theNameCacheOnce sync.Once
var theNameCache *docker.NameCache
var theNameCacheErr error

// generateNameCache generates a brand new *docker.NameCache.
func generateNameCache(cfg LocalSousConfig, ls LogSink, cl LocalDockerClient) (*docker.NameCache, error) {
	dbCfg := cfg.Docker.DBConfig()
	db, err := docker.GetDatabase(&dbCfg)
	if err != nil {
		return nil, errors.Wrap(err, "building name cache DB")
	}
	drh := cfg.Docker.RegistryHost
	return docker.NewNameCache(drh, cl.Client, ls.Child("docker-images"), db)
}
