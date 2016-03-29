# Alternatives to sous

This document provides a comparison between sous and other tools solving
similar problems, with the aim of helping anyone thinking about using
sous to make an informed decision.

Pull requests to correct or clarify any of these points would be very welcome.

See also our [rationale for building sous].

[rationale for building sous]: rationale.md

## Sous features

- Build
  - Local developer build
  - Operational builds
- Deployment
  - Global deployment state
  - Discovery and update of deployment to reflect declared global state

## Feature Matrix

| Tool        | Deploy state                         | Packaging        | Scheduling                     |
| ---         | ---                                  | ---              | ---                            |
| [Spinnaker] | automatic                            | VM image         | Cloud service                  |
| [PaaSTA]    | declarative                          | free-form Docker | Mesos/Marathon                 |
| [Nomad]     | declarative descs, imperative update | docker, VM, ...  | Nomad                          |
| [Swarm]     | imperative                           | docker           | is a scheduler, or Mesos or... |
| [Compose]   | describtive, no update(?)            | docker           | none - just docker             |
| Sous        | declarative                          | buildpack Docker | Mesos/Singularity              |
| [Otto]      |                                      |                  |                                |

## Spinnaker

Cluster and deployment management.

Cluster management is an abstraction over cloud services, simplifying the view
to a kind of least common denominator. This may or may not be useful at Open
Table, but is outside of the Sous scope.

Deploy management: a pipeline abstraction, which consists of a series of
stages. Arguably, it would be possible to build the automated component of Sous
as Spinnaker pipelines.

Would Spinnaker's pipelines re-introduce a configurability =~ divergence issue?

Spinnaker also appears to assume Jenkins - as far as I know, that would be a
new dependency. Maybe interfaces could be built for Igor?

Likewise, Rosco (the image Bakery) assumes GCE or AWS images as opposed to
Docker containers.

Ultimately the pipeline idea is a good one - does it make sense to have more
than one pipeline at OpenTable though? There's value to one-size-fits-all,
(convention over configuration etc), but there can be costs too.

## PaaSTA

Includes an excellent
[document](https://github.com/Yelp/paasta/blob/master/comparison.md)
Compares to further tools:
- ECS
- Kubernates
- Heroku
- Flynn
None of these address Mesos, the latter two don't use Docker.

Designed for containers and Mesos.

Uses Marathon and Chronos as opposed to Singularity.

No buildpacks.

Declarative cluster state.

Cluster design - PaaSTA considers clusters as an architectural entity.

Docker tags for part of configuration.

## Nomad

Cluster management, akin to Mesos (with a fuller feature set). Almost
completely agnostic about the built tools.

Multi-DC, multi-region aware - tasks can be run across DCs or clouds without
caring where exactly they are. Scale can be (I think?) described on a
per-region basis, as opposed to e.g. deploying completely separate clusters.

Each job is essentially a Mesos request+deploy, written as a YAML(?) config
file. `nomad run <file>` creates/updates the description of the job with nomad,
which determines what it needs to do to make the description true. This
interface is a lot like what we've been thinking about for the GDM -
substituting `git push` for `nomad run`

## Otto

Otto is a development tool - "the successor to Vagrant." PaaS, but for the
developer's environment. It's necessarily opinionated, and its opinions (almost
certainly) vary from ours.

Otto orchestrates Vagrant to set up VM envs, and Packer to build things. It
mostly replaces parts of a developer's setup process. I don't see why any team
at OT wouldn't decide to use Otto, but I don't think Otto would be compatible
with the Sous design principles.

Packer abstracts over e.g. AWS setup vs. Dockerfiles to use a single
configuration to build many different kinds of packaging.

## Docker Swarm

Docker API compatible docker scheduler. `docker up`s happen "somewhere" in the
cluster. Provides scheduling or can be run on top of Mesos. More analogous to
Singularity than to Sous.

## Docker Compose

Take a description file of a multi-container app, boot up all the described
containers. Recommended use cases are development and testing. Notably, I don't
see any facility for making smooth updates to a composed environment - changing
the Composefile and having that be reflected effectively entails bouncing the
cluster.

Swarm + Compose does *resemble* a PaaS, but Swarm is early days and Compose is
mostly targeted at development activities at the moment.

# Observations

## Context

Sous comes into being at a particular time at OpenTable.
We've already established expertise and dependencies on Mesos, Singularity and Docker.
We've built FrontDoor and Discovery.
These are very much the components from which a platform service is built from.
As a result, many potential alternatives to Sous would involve replacing these known components.
For instance, Kubernates would replace Mesos and Discovery at a bare minimum.
Interestingly, Yelp appears to have found themselves in the same position when they embarked on PaaSTA,
so at least we're in good company.

## Problems Addressed Elsewhere

There are a number of features of alternative tools that address problems in the platform service space.
At a bare minimum, these features stand as an example of problems other engineers have encountered
and determined were worth addressing.
It may not be that Sous needs to also address these issues, but a principled decision _not_ to seem meet.

Build pipelines are a integral part of the [Spinnaker].
We needn't provide the same kind of configurability -
in fact, that would be contrary to the consolidation goals of Sous.
However, the idea of build pipelines is important and possibly useful.
It may be worthwhile to consider a pipeline as we design the Sous build process.
For instance, it may make sense to provide some kind of lifecycle hooks along the way.
There's also the interesting concern of manual gating e.g. by QA staff.

It may be that the Sous pipeline will be the default outlet of whatever pipelines development teams develop
e.g. in TeamCity.

Nomad treats environments and location as a top level concern -
it seems worthwhile to examine that concern, and consider whether Sous is the right place to address it.

[Spinnaker]: http://spinnaker.io
[PaaSTA]: https://github.com/Yelp/paasta
[Nomad]: https://www.nomadproject.io
[Otto]: https://www.ottoproject.io
[Swarm]: https://github.com/docker/swarm/
[Compose]: https://github.com/docker/compose
