# sous [![Build Status](https://secure.travis-ci.org/opentable/sous.png?branch=master)](http://travis-ci.org/opentable/sous) [![Report card](https://goreportcard.com/badge/github.com/opentable/sous)](https://goreportcard.com/report/github.com/opentable/sous)

Sous is a tool for building, testing, and deploying applications, using
Docker, Mesos, and Singularity.

For contribution guidelines, see [here](./doc/contributions.md).

[View documentation in the doc/ directory.](https://github.com/opentable/sous/tree/master/doc)

# Using Sous

If you're looking to get started using Sous
to manage your service within a larger organization, read on.

## Installation

Sous is written in Go.
Once you have [Go set up on your machine,](./doc/setting-up-go.md)
you can install it by typing:

    $ go get -u -v github.com/opentable/sous

## Client Configuration

To use Sous effectively,
you'll need to know the URL of at least one running Sous server.
(If you're looking to set up a new Sous deployment,
see [this guide.](./doc/first-deployment-of-sous.md))
That URL will be particular to your organization,
but someone should be able to provide it to you.

Once you have it, run:

```
$ sous config server <URL>
```

(...replacing `<URL>` with the URL you were given.)

You should be good to go, but if you're curious,
client configuration is documented [here](./doc/client-config.md).

The **Installation** and **Client Configuration** steps
should only need to be done once on any given workstation.

## Hello sous

Now that you have a Sous client set up,
let's add a project to Sous management.

```bash
# Enter the directory of your project.
cd <my-project>

# Add a git tag with a semantic version:
git tag -a 1.2.3 && git push --tags

# Build the Sous-ready container:
sous build

# Let Sous know that the project exists:
sous init
```

At this point, Sous will
deploy 1 instance of your project everywhere it knows about.
You can limit this to a single Mesos cluster by
replacing the last command with `sous init -cluster <name>` -
Sous will provide a list of known clusters if you give it bad input.

## Day to day

From then on, the process is similar,
but you don't need to do the `sous init`
since Sous already knows about the project.

Instead, you'll want to use `sous deploy`, like this:

```bash
git tag -a 1.2.4 && git push --tags
sous build
sous deploy -tag 1.2.4 -cluster <name>
```

Note that these steps can be easily adapted to work
on continuous integrations servers, as well.

## Requirements

Sous shells out to your system to interact with Git and Docker. This is
a design decision, as it enables you to easily repeat the commands Sous
issues. That means that when they fail, as they sometimes do, you have
the power to re-play what happened, and figure out the issue.

You will need:

- Git >=2.2
- Go >= 1.7
- Docker >=1.10

On Mac, we recommend installing Docker by installing docker-machine
via the Docker Toolbox available at https://www.docker.com/toolbox

## LICENSE

MIT
