//+build !integration

package graph

import "github.com/opentable/sous/util/docker_registry"

func newDockerClient(ls LogSink) LocalDockerClient {
	return LocalDockerClient{docker_registry.NewClient(ls.Child("docker-client"))}
}
