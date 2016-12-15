package test_with_docker

import (
	"errors"
	"net"
	"time"
)

type (
	// An Agent manages operations directed at Docker
	// This is an interface that abstracts the differece between local
	// docker-daemons (useful, for instance, for Linux based CI (e.g. Travis)) and
	// VM hosted docker-machine managed daemons (e.g. for OS X development.
	Agent interface {
		//	ComposeServices uses docker-compose to set up one or more services, using
		//	serviceMap to check availability.
		//
		//	Importantly, the serviceMap is used both to determine if the services are
		//	already available - since docker-compose can take some time to execute, it
		//	can be handy to run the compose in a different console and let
		//	ComposeServices discover the services.
		//
		//	Finally, if ComposeServices determined that a service was missing and
		//	needed to be run, it will return a value that represents the
		//	docker-compose command that it executed. You can pass this value to
		//	Shutdown to shut down the docker-compose after tests have run.
		ComposeServices(string, serviceMap) (*command, error)

		// InstallFile puts a path found on the local machine to a path on the docker host.
		InstallFile(string, string) error

		// DifferingFile takes a list of pairs of [local, remote] paths, and filters them
		// for pairs whose contents differ.
		DifferingFiles(...[]string) ([][]string, error)

		// IP returns the IP address where the daemon is located.
		// In order to access the services provided by a docker-compose on a
		// docker-machine, we need to know the ip address. Some client test code
		// needs to know the IP address prior to starting up the services, which is
		// why this function is exposed
		IP() (net.IP, error)

		// MD5s computes digests of a list of paths
		// This can be used to compare to local digests and avoid copying files or
		// restarting the daemon
		MD5s(...string) (map[string]string, error)

		// RebuildService forces the rebuild of a docker-compose service.
		RebuildService(string, string) error

		// Shutdown terminates the set of services started by ComposeServices
		// If passed a nil (as ComposeServices returns in the event that all services
		// were available), Shutdown is a no-op
		Shutdown(*command)

		// ShutdownNow unconditionally terminates the agent.
		ShutdownNow()

		// RestartDaemon reboots the docker daemon
		RestartDaemon() error

		// Exec executes commands as root on the daemon host
		// It uses sudo
		Exec(...string) error

		// Cleanup performs the tasks required to shut down after a test
		Cleanup() error
	}

	agentCfg struct {
		timeout time.Duration
	}
	agentTrialF   func() agentBuilderF
	agentBuilderF func(agentCfg) Agent
)

var (
	agentTrials = []agentTrialF{dmTrial, ldTrial}
)

// NewAgent returns a new agent with the DefaultTimeout
func NewAgent() (Agent, error) {
	return NewAgentWithTimeout(DefaultTimeout)
}

// NewAgentWithTimeout returns a new agent with a user specified timeout
func NewAgentWithTimeout(timeout time.Duration) (Agent, error) {
	for _, tf := range agentTrials {
		if bf := tf(); bf != nil {
			return bf(agentCfg{timeout: timeout}), nil
		}
	}
	return nil, errors.New("Couldn't determine what the docker environment was to start an agent")
}
