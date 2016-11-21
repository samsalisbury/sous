# sous [![Build Status](https://secure.travis-ci.org/opentable/sous.png?branch=master)](http://travis-ci.org/opentable/sous) [![Report card](https://goreportcard.com/badge/github.com/opentable/sous)](https://goreportcard.com/report/github.com/opentable/sous)

Sous is a tool for building, testing, and deploying applications, using
Docker, Mesos, and Singularity.

For contribution guidelines, see [here](./doc/spinning_up.md).


[View documentation in the doc/ directory.](https://github.com/opentable/sous/tree/master/doc)

## Installation

Sous is written in Go. If you already have Go set up on your
machine, and have your GOPATH set up correctly, you can install it by
typing

    $ go get -u -v github.com/opentable/sous

## Client Configuration

To view (or create) your sous config, run:

    $ sous config

If a configuration file is not found, one will be created in ~/.config/sous/config.yaml

Client configuration is documented [here](./doc/client-config.md).

## Hello sous

A configured sous client can interact with an existing sous server using the [getting started](./doc/getting_started.md) file.

## Server Configuration

Placeholder for a link to server configuration documentation.

## Requirements

Sous shells out to your system to interact with Git and Docker. This is
a design decision, as it enables you to easily repeat the commands Sous
issues. That means that when they fail, as they sometimes do, you have
the power to re-play what happened, and figure out the issue.

You will need:

- Git >=2.2
- Go >= 1.6
- Docker >=1.10

On Mac, we recommend installing Docker by installing docker-machine
via the Docker Toolbox available at https://www.docker.com/toolbox

## LICENSE

MIT
