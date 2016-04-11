# Architecture

A collection of single-approaches, domain modelling, and overall structure of the Sous project.

Please read the [sous ontology] first, as it contains definitions for many of the words used in this document.

[sous ontology]: ontology.md

## Components

**Sous Build** is a tool used both for local development and on build servers.
It makes use of buildpacks to convert source code into a production container image.
There are a number of steps involved in this process.
The key requirement, beyond producing the container image, is labelling it with the revision of the source code from which it was derived
and registering it with the container image registry.

The **Sous Server** manages the Global Deploy Manifest (GDM).

The GDM is made up of service manifests.
The overall GDM is maintained as a versioned history, so that changes can be tracked over time.

A service manifest is identified with a canonical package name that points to a location within a source code repository.
For each target deployment cluster, it contains a deployment definition, which specifies

- the particular source code revision,
- environmental configuration, and
- a _resource declaration_

for deployment to the cluster in question.

Sous Server receives proposed updates to the GDM,
validates them using a series of automated checks
and commits the new GDM as the "intended" state.

Sous Server continuously compares the GDM to the real world, represented as an **actual deploy manifest**, or **ADM**.
The ADM is produced by interrogating all the known clusters for their actual deployment state.
The server then computes and issues commands to transform the ADM to match the GDM.
Once those commands have been carried out, and the ADM matches the GDM,
the new GDM is marked as "current" and "achieved"
and the previous "current" GDM version loses "current" but retains an "achieved" flag.

The server maintains a queue of GDM updates.
If there are items in the queue when a new update is received, the behavior depends on
the nature of the update
and the updates already in the queue.
If there aren't any updates in the queue for the same manifest already
(i.e. no updates that refer to the same piece of software)
the update is accepted into the queue, and the deployer is notified of their position in the queue.
If there are are updates with the same service, the update is rejected _a la_ a failed HTTP conditional update.
This should only ever happen if a service team tries to make multiple deploy updates at the same time, for the same software.
This type of conflict is assumed to be rare, so the incidence of rejected deployments should be small.

**Buildpacks** are sets of instructions used to build container images from source code.
The bare minimum buildpack starts from an existing Dockerfile, builds the associated container image and labels it for use by Sous Server.
More featureful buildpacks build an intermediate container as a host to produce deployment artifacts (e.g. a JAR file, a node_modules tree)
and then transfer that artifact into a deployment container image prepared to execute the artifact.
Also possible would be support containers e.g. to be test the resulting container by doing end-to-end tests.

A buildpack should be able to evaluate a working tree containing source code and report whether it knows how to generate an artifact from that context.
Sous Build uses this facility to enumerate the candidate buildpacks for a project,
and possibly auto-select the single reporting buildpack.
If no single buildpack presents itself, the developer can select from those that do report compatibility
or inquire via the `sous` tool as to why certain buildpacks have rejected their project.

**Contracts** are simple, well structured tests of the behavior of a container, in terms of interactions with other containers.
They're analogous to unit tests, where the unit of execution is a container.
Contracts can also be seen as assertions on the state of the system as a whole.
The current challenge is to provide either/both of
 mocks or real instances for dependencies of the service;
  this includes other services and not-yet-sousified services like databases or message queues
 and
 setup/teardown of real instances of those service dependences.
In the latter case, the contracts are more properly considered integration tests.

## Deployment Descriptions

Every application deployed by Sous corresponds to a deployment description.
These descriptions are a concept internal to the Sous system;
users will usually manipulate applications or instances, which are each views over deployment descriptions.
(Deployment descriptions are sometimes referred to simply as 'deployments.')

A deployment description is a tuple which binds:
- A cluster
- A set of dependant resources
- A [named version](#named_versions)
- A dictionary of environment variables
- A list of project owners
- An application kind
- The number of instances the application should reside in

Not only can a particular deployment description refer to multiple running application instances (by virtue of the instance count),
deployment descriptions can refer, for instance to intended states which haven't been realized yet.

Generally speaking, deployment descriptions have two sources:
they can be built from the deployment commands issued by users
or
they can be synthesized from data collected from the running clusters.

"Intended" deployments (that is, those build as the result of user commands)
exist in a number of states -
waiting,
current,
achieved (but no longer current),
passed over (older than current, but was never itself current).
These states are determined as the server checks the state of the running clusters and issues commands to update them.

Deployments are also frequently organized into deployment sets.
The criteria by which a deployment set is selected from all possible deployments is used to describe the set.
The most basic sets are
the _actual_ set (all the deployments collected from running clusters),
and
the _current_ set (all the deployments that are current i.e. match the last known state of clusters).
The ADM is represented internally exactly by the actual deployment set,
and
the GDM is represented internally by the current deployment set.
Other sets include the set of deployments in a particular cluster,
or
the historical deployments of a particular service, as represented by a canonical package name.


## Named Versions ##

Named Versions are used to identify a number of items related to a specific version of a piece of software.
They consist of a triple of a source repository, a version identifier and a path.

The source repository is a URL, typically to a git repository. For example:
`https://example.com/gitproject.git`
or
`git://example.com/project.git`
.
While in theory other kinds of source URL might be contemplated, git is the only kind contemplated at the moment.

The version identifier consists of both an absolute revision ID,
	and a human-readable tag, bound to that revision ID.
The tag should be a [semantic version].

The version identifier is used to identify corresponding container images, using build metadata.

A common manifestation of the version identifier is in docker image names.
Here, it is transformed into another semantic version by
	affixing the revision ID to the version using the semver metadata field
	(i.e. text after a '+').

It is used to identify a precise, unique version of the software.

Note: in practical terms, "revision ID" means the git commit SHA.
If other kinds of source are to be defined, the unique identifier requirement would need to be satisfied.

[semantic version]:(http://semver.org/)

(As a practical matter, the actual version would be used as a tag to trigger builds in Sous -
that's an interaction with the entity name definition, not a part of it.)

Finally, the path is used to specify a particular entity to be found within the source.
For instance, a single directory within a git repository.

### Checking Names

The use of version number plus unique id over-specifies the entity in question.
Where the name is being used in an interactive context,
	and the version number no longer matches the recorded revision ID
		(i.e. the git tag has been moved)
	an error should be reported and the operation should be rejected.
However, where name is being used in a batch context,
	the revision ID (e.g. the git SHA) should be used, and considered correct,
	but if the corresponding version tag has been moved,
	the user should be notified.

### Canonical Package Names

To refer to a piece of software over time, independent of a particular version in its evolution,
it's possible to use just the source URL and path components of the appropriate entity name.
	This pair is known as the "Canonical Package Name".
This will be appropriate in particular contexts, such as to identify manifests,
	and always mutually exclusive to the use of the fully qualified entity name.

### String Representations

While named versions are triples of values, they must sometimes be represented and manipulated as strings.
Specifically, at the interfaces of Sous, both with human beings and other software.

The default string representation of an entity name begins with a character not matching `[A-Za-z0-9]`
and which doesn't appear any of the three parts of the name.
This character will be used as the delimiter for the representation.

The rest of the representation is straightforward -
concatenate the source URL,
the delimiter,
the version identifier,
the delimiter,
and then the path.

For example:
`^git://example.com/project.git^v1.0.0-rc+132984adf^/my-app`

When ',' is a legitimate choice as the delimiter, it should be preferred, and it may be omitted from the first position in the string.
If the first character in an entity name string is alphabetic, the delimiter should be taken to be ','.

Using the default delimiter rule:
`git://example.com/project.git,v1.0.0-rc+132984adf,/src/package.json`

### Opaque Representations

In some contexts, the components of an entity name may not be acceptable.
For instance, Docker image names treat '/', ':' and '+' specially.

For these uses, the opaque representation exists.
To produce an opaque representation, begin by
choosing any string, excepting that it cannot contain 'sous' as a substring.
 (the string should be chosen to suggest to a human the entity's identity,
 and in order to disambiguate the resulting string for e.g. tab completion.)
Concatenate the string with 'sous',
	generating the usual string representation.
	Base64 encode the representation.
	Concatenate the encoded representation.

In general, opaque representations should be useful as exactly that, but note that the original entity name can be recovered from them if needed.
