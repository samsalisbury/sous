# sous rationale

All software projects require justification for their existence. Sous is
no different, and there are many valid questions to be asked about why
OpenTable has decided to invest in this kind of project. We believe that
it is worth investing in answering those questions, and in providing
all the necessary background information, so you can make an informed
choice about whether it's sous, or some other project that fits your
needs best.

## FAQ

- [Why do we need this?](#why-do-we-need-this)
- [Why not use a pre-existing tool?](#why-not-use-a-pre-existing-tool)

### Why do we need this?

Deployment is only one part of the picture. Right now at OpenTable,
different teams have different ways of deploying, and there is no way to
easily track what was deployed where, when, and by who. We do not have a
good enough handle on what we have deployed globally, in more ways than
one.

We believe that the global state of deployed applications is a valuable
artefact in its own right, to enable us to version control "OpenTable" as
a whole, at least to some degree. Right now, we don't have a good way of
doing this, and that's one of the main features sous will bring.

We are a large and growing engineering team, with 34 separate teams,
and rising. We have scaled up hugely over the past few years, and the
amount of diversity in our platform has followed this growth. Diversity
is great for individual software engineering teams, being able to pick
the best tools for the job; however it can make life very difficult
for operations trying to keep software up and running, and for the
engineering organisation more broadly. Sous aims to address some of this
diversity by at least providing standardised container base images, and a
standard build and deploy pipeline.

### Why not use a pre-existing tool?

There are many other tools in this space, which are intended to tackle
similar problems to that which we are tackling with sous. For example:

- [Spinnaker] by Netflix
- [PaaSTA] by Yelp
- [Nomad] by Hashicorp
- [Otto] by Hashicorp
- [Swarm] + [Compose] by Docker

[Spinnaker]: http://spinnaker.io
[PaaSTA]: https://github.com/Yelp/paasta
[Nomad]: https://www.nomadproject.io
[Otto]: https://www.ottoproject.io
[Swarm]: https://github.com/docker/swarm/
[Compose]: https://github.com/docker/compose

It is possible that some combination of these tools may also be able
to solve the problems we are trying to solve with sous. However, after
looking into a number of them, we have not yet found one that fits our
needs exactly. It may be possible to modify or write plugins for some of
the above but we believe that by writing our own tool, we will be able
to solve all of the problems we're trying to solve in a unified and easy
to understand way; and more importantly in less time, than it would take
to wrestle one of these tools to do our bidding.

We have an established ecosystem using Mesos, Docker, and Singularity
as our scheduler. These technologies are now well understood by the
team, and we do not want to lose the benefits they bring, as well as the
knowledge we have about them by changing any of these in the short term.
None of the tools we have seen has been designed to work with this exact
stack.

We do not believe in drastic, immediate change to the core fabric of
our deployment environments, as that would be certain to be followed
by a period of instability. Therefore, we are building Sous to corral
our existing infrastructure into a smarter platform, that requires less
boilerplate to use. We also want to decouple our applications from
their deployment environment as much as possible, giving us the option
of switching from Singularity to Marathon, or Mesos to Kubernetes, or
Docker to Rocket, at some point in the future. If every team had their
own custom deployment pipeline, we would not be able to make such a
switch without invoking the cooperation of every team in the company.
So this is why we don't want to switch out our whole deployment fabric
just to suit a tool like Spinnaker, PaaSTA, or Swarm, as it would be too
risky.

Please see the [deployment tools feature matrix] for a quick comparison
of these tools.

[deployment tools feature matrix]: alternatives.md#feature-matrix
