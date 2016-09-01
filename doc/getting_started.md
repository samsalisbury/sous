# Getting Started

One of the primary goals of Sous is that
applications developers should need to adapt their projects
as little as possible to the Sous deployment process.

To greater or lesser extents, we feel we've been successful.
There are, certainly, further steps towards this goal.
Furthermore, it will probably be impossible for Sous to be
absolutely agnostic about the kinds of services that it deploys.

It may help to review
[the deploy process](./deployment-workflow.md).

# Assumptions

Let's try to address the assumptions that Sous makes about your project.

## Networked Service

First of all, Sous assumes that you're building an independent networked service.
It may work to use it for other kinds of software,
but the intent is to deploy programs that listen for requests and service them.

## Singularity

Sous (currently) only deploys into Singularity clusters.
There are assumptions about
how your application works
that it therefore carries over
from Singularity.

More specifically,
we assume bridged networking mode in Singularity.
You are provided with two environment variables you must use to configure your server:

- `TASK_HOST` contains the host name of the node you're running on, and
- `PORT0` contains the number of an available port you must listen on.

If you asked for more than one port in your resources configuration, you will also have
    `PORT1, PORT2, ... PORTN-1` also indicating additional free ports you can listen on.

You must configure your server to listen on the IP address `0.0.0.0` (i.e. all IP addresses)
The `TASK_HOST` environment variable is the hostname outside traffic will need to use
to find your app.

While there are many many options available to configure Singularity deployments,
Sous currently provides a limited subset of these options, which satisfy OpenTable's
current requirements.
For instance, automatic port mapping is not currently available, since at OpenTable,
we require each application to announce its own externally routable address.
Therefore the information provided by the `TASK_HOST` and `PORT0` variables is canonical,
and must be respected by the application.
Automatic port mapping would significantly complicate this model.

## Docker

Sous assumes that applications build into docker containers.
There are a number of assumptions that Docker imposes,
which Sous carries over.

One of the assumptions Sous presently makes is that
your application will provide its own working Dockerfile.
The docker image produced by a simple `docker build .` must represent your application.
We plan to provide support for other build mechanisms in the short to medium term.

# Summary

Consequent to the above,
applications intending to be deployed via Sous
should satisfy the following requirements:

* Have a Dockerfile that creates an image that runs correctly when run with
`docker run -e PORT0=12345 -e TASK_HOST=somehost <image name>`.
* Binds to the http address `0.0.0.0:$PORT0`
* Announces its outside address `$TASK_HOST:$PORT0``
* Application configuration that varies with environment should be read from environment variables.
