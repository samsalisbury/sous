package git

import (
	"testing"

	"github.com/nyarly/testify/assert"
)

func TestRemoteProcessing(t *testing.T) {
	assert := assert.New(t)

	rs := Remotes{
		`origin`: Remote{
			Name:     `origin`,
			FetchURL: `https://github.com/origin/fetch`,
			PushURL:  `https://github.com/origin/push`,
		},
		`upstream`: Remote{
			Name:     `upstream`,
			FetchURL: `https://github.com/upstream/fetch.git`,
			PushURL:  `https://github.com/upstream/push.git`,
		},
		`something`: Remote{
			Name:     `something`,
			FetchURL: `git@github.com:something/fetch.git`,
			PushURL:  `git@github.com:something/push.git`,
		},
	}

	assert.Equal(`github.com/upstream/fetch`, guessPrimaryRemote(rs))
	afs := allFetchURLs(rs)
	assert.Contains(afs, `github.com/origin/fetch`)
	assert.NotContains(afs, `github.com/origin/push`)
	assert.Contains(afs, `github.com/upstream/fetch`)
	assert.NotContains(afs, `github.com/upstream/push`)
	assert.Contains(afs, `github.com/something/fetch`)
	assert.NotContains(afs, `github.com/something/push`)

	delete(rs, `upstream`)
	assert.Equal(`github.com/origin/fetch`, guessPrimaryRemote(rs))
	afs = allFetchURLs(rs)
	assert.Contains(afs, `github.com/origin/fetch`)
	assert.NotContains(afs, `github.com/upstream/fetch`)

	delete(rs, `origin`)
	assert.Equal(``, guessPrimaryRemote(rs))
	afs = allFetchURLs(rs)
	assert.NotContains(afs, `github.com/upstream/fetch`)
	assert.Contains(afs, `github.com/something/fetch`)
}
