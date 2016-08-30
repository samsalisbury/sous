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
we assume the default networking mode in Singularity:
you'll be provided with two environment variables you must use to configure your server:
    `TASK_HOST` contains the host name of the node you're running on, and
    `PORT0` contains the number of an available port you must listen on.
If you asked for more than one port in your resources configuration, you will also have
    `PORT1, PORT2, ... PORTN-1` also indicating free ports you can listen on.
You must, however, listen on the IP address `0.0.0.0` (i.e. all IP addresses)
The `TASK_HOST` environment variable is the hostname outside traffic will need to use
to find this app.


While there are many many options about how Singularity works,
Sous provides a limited subset of these options.
For instance, automatic port mapping is not available,
on the grounds that a Singularity based service
will need to announce itself somehow,
and will need to know its outside port regardless.
In the default networking mode, the outside and inside ports are the same.

## Docker

Sous assumes that applications build into docker containers.
There are a number of assumptions that Docker imposes,
which like Singularity,
Sous carries over.

One of the assumptions Sous presently makes but which is
prioritized for correction,
is that your application will provide its own Dockerfile.

# Summary

Consequent to the above,
applications intending to be deployed via Sous
should address the following points:

* The binary artifact is a Docker container, described by a Dockerfile you write.
* All configuration for the application should come through environment variables.
  * One pattern to address this requirement is to build "flavors" of configuration, and select based on ENV
* You'll need to get your network addresses from `ENV[PORT0]` and bind accordingly.
