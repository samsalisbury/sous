# Architecture

A collection of single-approaches, domain modeling, and overall structure of the Sous project.

## Components

**Sous Build** is a tool used both for local development and on build servers.
It makes use of buildpacks to convert a code artifact into a production container.
There are a number of steps involved in this process.
The key requirement, beyond producing the container, is labelling it with the pinned name of the source code from which it was derived
and registering it with the deployment container registry.

The **Sous Server** manages the Global Deploy Manifest.

The GDM is made up of service manifests.
The overall GDM is maintained as a versioned history, so that changes can be tracked over time.

A service manifest is identified with an artifact name that points to the source project that it represents.
It contains a pinning that indicates the particular source code (intended to be) in production.

Sous Server receives proposed updates to the GDM,
validates them
and commits the new GDM as the "intended" state.
It then compares the new GDM to an "actual manifest" as produced by producing a snapshot of the running Singularity deploy.
The server then computes and issues commands to transform the actual manifest into the GDM.
Once those commands have been carried out, and the new Actual Manifest matches the intended manifest,
the new GDM is marked as "current" and "achieved"
and the previous "current" GDM version loses "current" but retains an "achieved" flag.

_Discussion:
It is possible that if multiple steps of "intended" manifest state were received
that intermediate states might not ever have been achieved.
Alternatively, proposed updates might be rejected while the deployment state is in flux.
Or the new proposed GDMs might be queued and walked up, achieving each in turn until no new intended states exist._

_The drawback of reject-in-flux is the perceived friction introduced into the Sous process.
Conversely, queue-and-walk might fail at some point, in which case later intended states would need to be treated as failed, and there's a problem of notification._

_One possibility would be to treat services as the versionable entities, and the global "current" and "acheived" states to be sets of particular versioned service manifests._

**Buildpacks** are sets of instructions to build containers.
The bare minimum buildpack starts from an existing Dockerfile, builds the associated container and labels it for use by Sous Server.
More featureful buildpacks build an intermediate container as a host to produce deployment artifacts (e.g. a JAR file, a node_modules tree)
and then transfer that artifact into a deployment container prepared to execute the artifact.
Also possible would be support containers e.g. to be test the resulting container by doing end-to-end tests.

A buildpack should be able to evaluate a source artifact and report whether it can process that artifact.
Sous Build uses this facility to enumerate the candidate builders for a project,
and possibly auto-select the single reporting buildpack.
If no single buildpack presents itself, the developer can select from those that do report compatibility
or inquire via the `sous` tool as to why certain buildpacks have rejected their project.

**Contracts** are simple, well structured tests of the behavior of a container.
They're analogous to unit tests, where the unit of execution is a container.
The current challenge to provide either/both of
 mocks for dependencies of the service;
  both other services and not-yet-sousified services like databases or message queues
 and
 setup/teardown of real instances of those service dependences.
In the latter case, the contracts are more properly considered integration tests.


## Artifact Names ##

Artifact names are used to identify a number of code artifacts,
and are exactly a pair of a git repository URL (ignoring the schema)
and a path within the repository
that identifies the canonical source of the artifact.

A pinned artifact name is an artifact name
paired with a commit digest/tag-name pair
which both precisely identifies a specific instance of the artifact as well as providing a semantically useful name for it.

Artifact names (pinned or not) are used to identify source code for deployments, contract definitions and buildpacks.

Artifact names can be represented by a string like
`github.com:project.git,/components/service`
and pinned artifact names like
`github.com:project.git,/components/service;v1.0.0,cabbagedeadbeef`.
Pins
(`v1.0.0,cabbagedeadbeef`)
can sometimes be provided in the context of an artifact name in order indicate the associated pinned artifact.

_Open question:
using , and ; as delimiters implies that they're forbidden in the delimited text - what if a subdir or tag name uses those characters?_

_Alternative:
treat these complex data types as exactly that and use dictionaries where ever they appear.
This will make CLI UIs somewhat more cumbersome,
and admit the failure case of (temporarily) inconsistent state when one k/v in the dictionary changes out of sync with the rest._

_Open question:
tying tag and revsha together leads to this problem: what if a tag is republished?
The git manual strongly implies that public tags shouldn't be republished, but like many things you shouldn't do, git figures you know best.
How does Sous handle the case where a known pinned artifact name is no longer accurate because the named revision isn't the tag anymore?_
