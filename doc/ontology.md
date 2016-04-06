# sous ontology

Sous is the glue which integrates all of the components necessary to deploy software.
This means it deals with a broad range of domain objects.
In order to avoid confusion, here we define the main domain object types that Sous deals with.

All documentation and source code that deals with the concepts defined below should use these names, or derivatives thereof.

- **software** (aka **a piece of software**, **service** or **application**) is a single logical unit of execution,
	achieving a specific set of goals for an organisation.
	A single piece of software may have multiple different _versions_, as described by the _repository_ in which it lives.
- **version** (aka **software version**) is a particular version of a piece of _software_,
	which should broadly achieve similar organisational goals as other versions of the same software.
- **source repository** (aka **repo**, or just **repository**) is a [versioned repository] of source code.
	A repository must define at least one complete piece of _software_,
	it may define multiple pieces of software, but at most one piece of software per sub-directory.
- **source code** (aka **codebase**) is a source code _repository_ or a sub-tree of a repository which defines a single piece of _software_.
- **canonical package name** is a name which completely identifies the location of _source code_ for a single piece of _software_.
	This is the repository in which it lives paired with the sub-directory inside that repository which contains the source code for a single piece of software.
	It _does not_ include any _revision_ information.
- **revision** (aka **source code revision**) is a specific _version_ of the source code inside a repository.
	A revision ought to unambiguously identify the exact contents of a _working tree_ at a particular point in time.
- **repository tag** (aka **tag** or **annotated tag**) is a human-readable identifier for a specific _revision_ inside a _repository_.
	Once published, a repository tag should never be made to point at any other revision.
- **working tree** is a context within a checked-out manifestation of a specific _revision_ of a repository.
	It includes the sub-directory within that repository that the _user_ is currently in.
- **user** is the person or other agent which directly invokes sous commands.
- **artifact** refers to a compiled instance of a specific _revision_ of _source code_.
	An artifact must be a complete description of the software, minus _configuration,_ which can be placed inside an _image_ to run the _software._
- **configuration** refers to any configuration which may vary per _deployment_.
	As a noun, "configuration" usually means the complete set of configuration values needed to run a _container_ inside a specific _cluster_.
- **resource** is any external resource that some _software_ needs in order to run.
	Typically this means a quota of a specific type of commodity hardware (memory, CPU cycles, disk space, etc.),
	but may also be used to represent a specific named instance of something in future.
- **resource declaration** is a list of quantified _resource_ requirements for a particular _deployment_ of _software._
- **image** (aka **container image**) is a ready-to-run image containing an _artifact_ which executes the software defined by the _source code_ inside a _container_.
	An image _should not_ contain any _configuration_, nor any _resource declaration._
- **image registry** (aka just **registry**) is a server which stores _container images_.
- **container** is an executing (or paused) [software container], created by invoking an _image_ along with a specific _configuration_.
	A container may be run on a developer's machine, on a CI server, or inside a _cluster._
	Inside a cluster, a container usually requires a _resource declaration_, in addition to a configuration, in order to run.
- **cluster** is a compute cluster of some kind, representing a single deployment target.
	A cluster contains a set of _resources_ which it makes available to running _containers._
	Typically, a cluster will be located inside a single datacentre, in a single region, and have a single purpose, e.g. "production-uk", "staging-uk", "qa-uswest2"
- **deployment** is 4-tuple containing
	- a specific _revision_ of a piece of software
	- inside a named _cluster_,
	- along with a _configuration_ and
	- a _resource declaration_
- **manifest** (aka **deployment manifest**, or **service manifest**) is a pairing of a _canonical package name_ with a set of _deployments_.
- **global deploy manifest** (aka **GDM**) is the set of all manifests (which must have unique _canonical package names_) deployed and managed by a single **sous server**
- **actual deploy state** (aka **ADS**) uses the same data structure as the _GDM_, but populated with a representation of the real-world deployment state.

[process]: https://en.wikipedia.org/wiki/Process_(computing)
[software container]: https://en.wikipedia.org/wiki/software_container
[versioned repository]: https://en.wikipedia.org/wiki/Repository_(version_control)

