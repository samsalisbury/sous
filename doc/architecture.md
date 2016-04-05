# Architecture

A collection of single-approaches, domain modeling, and overall structure of the Sous project.

## Components

**Sous Build** is a tool used both for local development and on build servers.
It makes use of buildpacks to convert a source code into a production container.
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

The server maintains a queue of GDM updates.
If there are items in the queue when a new update is received, the behavior depends on
the nature of the update
and the updates already in the queue.
If the there aren't any updates in the queue for the same service already
(i.e. no updates that refer to the same entity family)
the update is accepted into the queue, and the deployer is notified of their position in the queue.
If there are are updates with the same service, the update is rejected _a la_ a failed HTTP conditional update.
This should only ever happen if a service team tries to make multiple deploy updates at the same time,
so it indicates a communications issue in a small team.

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

## Entity Names ##

Entity names are used to identify a number of generalized items within Sous,
and consist of a triple of a source, a version identifier and a path.

The source is a URL, typically to a git repository. For example:
`https://example.com/gitproject.git`
or
`git://example.com/project.git`
.
While in theory other kinds of source URL might be contemplated, git is the only kind contemplated at the moment.

The version identifier is a [semantic version], where the build metadata
(i.e. text after a '+')
is used to identify a precise, unique version of the resource identified by the URL.
In practical terms, this means the git commit SHA.
If other kinds of source are to be defined, the unique identifier requirement would need to be satisfied.

[semantic version]:(http://semver.org/)

(As a practical matter, the actual version would be used as a tag to trigger builds in Sous - 
that's an interaction with the entity name definition, not a part of it.)

Finally, the path is used to specify a particular entity to be found within the source.
For instance, a single file within a git directory.

### Resolving Names

The use of version number plus unique id over-specifies the entity in question.
Where the name is being used in an interactive context, an error should be reported and the operation should be rejected.
However, where name is being used in a batch context, the unique id (e.g. the git SHA) should be used as correct,
and the disparity should be reported via a notification.

### Entity Families

To refer to an entity over time, independent of a particular moment in its evolution, 
it's possible to use just the source URL and path components of the appropriate entity name.
This will be appropriate in particular contexts, 
and always mutually exclusive to the use of the fully qualified entity name.

### String Representations

While entity names are triples of values, they must sometimes be represented and manipulated as strings.
Specifically, at the interfaces of Sous, both with human beings and other software.

The default string representation of an entity name begins with a character not in the range A-Z or a-z
and which doesn't appear any of the three parts of the name.
This character will be used as the delimiter for the representation.

The rest of the representation is straightforward -
concatenate the source URL,
the delimiter,
the version identifier,
the delimiter,
and then the path.

For example:
`^git://example.com/project.git^v1.0.0-rc+132984adf^/src/package.json`

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
Concatenate the string 'sous'.
Generating the usual string representation.
Base64 encode the representation.
Concatenate the encoded representation.

In general, opaque representations should be useful as exactly that, but note that the original entity name can be recovered from them if needed.
