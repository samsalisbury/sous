# Build Strategies

*This document will be an explanation of Build Strategies and how they work.
For the time being,
it documents our design process and ideas.*

## Design Goals

`sous build` should:

* Build using a Docker image
  (known as a "builder image")
  configured in the manifest.
* Produce runnable Docker images as output
  (known as "product images")
* Cache fetched dependencies when building on developer machines
  (e.g. Maven's `.m2` or node's `node_modules` or Ruby gems).
* Cache intermediate build products when building on developer machines
  (for rapid local development.)
* Perform a clean build at will
  (for continuous delivery).
* Produce the smallest possible product image layer
  (i.e. the part that represents the application being built
  rather than its runtime dependencies).
* Produce more than one product image per source code repository.
* Users should be able to make small overrides to the build process
  locally on their machine
  to facilitate experimentation.
* It should be possible to audit the build images in use
  so that they can be consolidated.
* `sous init`, or a related tool,
  should be able to guess at
  the "best" named build configuration and set it up automatically,
  or provide hints to the local operator about what to choose.
* `sous build` should hint at the above tool
  when a configuration is unavailable or sub-optimal
  (i.e. at some point the "simple-dockerfile" strategy
  might suggest using "mount-run-split".


## Current Proposal

Each manifest has a `Build` stanza describing how sous should build its images.
```
  Build:
    Strategy: mount-run-split
    Image: docker.internal.com/our-maven:latest
```

Possible `Strategy` values could be:

* `mount-run-split` meaning
  we run the `Image` with various directories from the local machine mounted.
  The result is that one of those directories
  is populated with built runnable artifacts.
  We then split those artifacts
  amongst various output containers ready for deployment.
* `split-build` meaning
  we build the provided `Dockerfile`
  (conceivably synthesized as descibed below for `mount-run-split`
  and extract files from the resulting image
  in a way similar to `mount-run-split`.
* `simple-dockerfile` meaning
  we simply build the `Dockerfile`
  in the project's source code repository (actually `SourceLocation`).
  This `Strategy` would not use the `Image` field at all
  and would is capable of producing only one artifact at a time
  (the docker image produced by `docker build`).

Simple-dockerfile strategy and
"split" builds are already supported by Sous,
and are currently the default when a Dockerfile is present in the working directory
when executing 'sous build'.
(The selection depends on whether the Dockerfile or its parent image
defines a "magic" environment variable.)

It is our intention to deprecate
both of these strategies in favor of `mount-run-split`
once it becomes available.

### mount-run-split builds

We start by synthesizing a one-line Dockerfile in memory
based on the image named in `Build.Image` from the YAML proposed above.

`Example synthetic dockerfile`:
```
FROM docker.internal.com/our-maven:latest
```

`sous build` builds this image, known as the "builder image", using `docker build`.
We call it the "builder image" because
it is used to build the project,
and is not itself a deployable artifact.
(We could simply `docker pull` the image,
but we get the advantage
of having a single code path for one-off experiments,
see below.)

We then `docker run` the built image with four mounted volumes:

* `/input` for the source (i.e. code and other things checked into the repo)
* `/vendor` for externally fetched dependencies
* `/working` for intermediate products
* `/output` for output products (e.g. jar files, directories, executables etc.)

Sous is responsible for making a copy of files from the current sous context,
and copying them into the `/input` directory inside the container.

The `/output` directory will be a temporary on the sous user's machine,
mounted inside the container. A new `/output` dir will be created per build.

The `/vendor` and `/working` directories will be mounted from the local filesystem,
we anticipate them being under a Sous-specific subdirectory of
`$XDG_DATA_HOME`
or
`$XDG_CACHE_HOME`
(i.e.
`~/.local/share/sous`
and
`~/.cache/sous`.)

Directories should be created for each unique project,
although arguably the `/vendor` directory should
be per build image,
and let the image code sub-partition the directory.
(e.g. Maven likely gets maximal benefit out of a single `/vendor` directory.)

The builder images will be responsible
for arranging the volumes into the directory structure
required by the build tools
(e.g. symlinks or `mount -o bind ...` for `~/.mvn2`).

Once the `docker run` terminates,
at the root of the directory mounted on `/output`
Sous will look for a file called `runspec.json`.
It is a build failure for this file to be missing.
Generally, the build process in
the image
will generate this "Runspec",
but it's not uncommon for a static file
to simply be copied into place.

A runspec looks like this:
```json
{
  "images": {
    "service": {
      "image": {
        "type": "Docker",
        "from": "microsoft/aspnetcore:2.0"
      },
      "files": [
        {
          "source": {"dir": "/srv"},
          "dest": {"dir": "/"}
        }
      ],
      "exec": ["dotnet", "/srv/service.dll"]
    }
  }
}
```

The `images` object is keyed by `offset` which corresponds to a subdirectory in the repo.
The `images` object can therefore contain up to one item per subdirectory in the repo (recursively).
Acceptable `offset` values include sub-sub-directories and so-forth, e.g.: `service` `src/service` `src/service/api` etc).

Sous will convert this runspec into
a series of synthetic Dockerfiles based on the runspec data, e.g.:

```Dockerfile
FROM microsoft/aspnetcore:2.0
COPY /srv /
CMD ["dotnet", "/srv/service.dll"]
```

It will `build` them in the directory mounted on `/output`.
Note that at this point all compilation has taken place,
and the 'build' should be merely a `COPY` operation.

The resulting images will be labelled with Sous metadata,
with the offset pulled from the name of the image object
(in this case: `service`.)

It is the responsibility of the build image
to produce at most one offset per subdirectory,
and to determine which subdirectories represent runnable items.

For example, Maven projects have the idea of "submodules"
defined in their pom.xml file, which correspond to
subdirectories. At present there is nothing inside the `pom.xml` that indicates
if a particular module corresponds to a runnable/deployable artifact.
We will need to solve this problem, some potential options are:

1. Require that any deployable modules are first `sous init`ed and check for the existence
   of a manifest for each.
2. Require a file in the root of the source repo listing deployable offsets.
3. Establish a rule that `pom.xml` files must add some metadata to the `modules` section about
   which offsets are to be considered deployable. (Other ecosystems would need an equivalent
   rule in this case.)

> I (Sam) favour option 1 as that ensures that we have deployment configuration for these
offsets as well, so that we can actually deploy them. 'sous init' needs to provide guidance
about 'sous init'ing offsets, and 'sous build' / 'sous deploy' need to print very clear messages
about each item they are building and deploying so that if any are missing they can be added
by the user.

> **Note** we could then allow recognising tags for differing offsets using tag prefixes to support
versioning offsets differently where necessary. In the absence of prefixed-tags we could assume that
the entire repo is versioned as one. We would probably need to be explicit about which model is 
used in the 'project' definition in the GDM.

### Observations

Because the build image will generally just be downloaded,
the `docker build` of the initial container will usually be (almost) a no-op -
the first time there'll be download of data,
and thereafter there'll be a check of the local daemon's cache of the build image.
However, using a Dockerfile for this purpose
allows for us to fulfill the "small changes" requirement.

It should be possible for Sous
to add labels related to the build process
to the final build images
for auditing purposes.

We could add a `/sous` directory to built deploy artifacts
and store e.g. the build Dockerfile there for reference purposes.

In order to allow for easy development of new build containers,
and to effect transition to existing ones,
`sous build` would be extended with two new flags:
`-dev` would allow a number of non-production-suitable changes
to the Sous build process, like
caching of intermediate artifacts,
flags to build programs to output more logging.
`-override-build-image <Dockerfile>`
would only be available in `-dev` mode,
(or simply imply it)
and would use the named Dockerfile
instead of consulting the manifest.

Right after getting the manifest to build from,
Sous should issue commands like
`git config sous.strategy mount-run-split`
and
`git config sous.image private.repo.org/main-nodejs-builder:latest`
to store
the strategy and
the image
that it discovers.
Then, if on a future build it can't get the manifest
(e.g. no network connection, Sous server is down)
it can consult the local git configuration
to recover that information.
A warning should be generated to
inform the user that this has happened, however.

At some point in the future,
we can convert to a notarized platform
where the build images would need to be notarized
in order to be used
(or at least "unsigned build image" becomes an advisory)
which would encourage/require
that updates to the build images be reviewed and approved.
In the meantime, the `Build>Image` fields on manifests
would form a catalogue of build images.

## Discussion

### Migration

At time of introduction, a missing or empty Build section should be treated as
a signal that we should look for a local Dockerfile
(i.e. fall back to the existing build strategy).
If there is an existing one-line Dockerfile,
`sous build` might make (or suggest)
this change itself.

We might have a period where we encourage users to switch
to having a Build section
even if it's to have `Strategy: simple-dockerfile`.
Then, we'd trust what we find in the manifest,
but fall back to looking for a Dockerfile.

Finally, we'd come to a point where we require the `Build` section,
and error out if it's missing.

`sous init` would need to be updated to default to adding a `Build`
before the final state.
Ideally, it should "guess" at an image up front,
or prompt with options and it's best guess.


### Comment and dialog

**SS:** Should we say “small-as-possible product-specific layers" instead? In
theory this works better with caching at the daemon level at least. But then,
we also want to avoid one super image with all dependencies on it. So, not sure
how to word this exactly, maybe leave it out as a requirement for now?

**JL:** The only thing I'm strongly against
is leaving the requirement out.
I think we can consider layers vs. image a *specification* detail -
but for the purpose of requirements,
I want the build images to
a) occupy as little space on disk as possible and
b) to take as little time to transmit
from artifact repository
to execution agents
as possible.
Overall small images,
and small unique layers both address those requirements.

**SS:** We should allow using an embedded `Dockerfile` in the `Build` stanza of
the manifest, so that small overrides can be made for specific projects.

**JL:** By limiting the manfiest entry to an image,
it means that "blessed" build images will need to be built and pushed
external to a particular project.
(Our internal set of build images being an example of that process.)
I don't want to start inlining Dockerfiles into the manifest -
that seems like an auditing nightmare.

**SS:** So for anything that wants to be deployed, you must push a build image that
can build it, experimentation is strictly off-line on local dev machines. Any builder
image in your source code repository will be ignored unless using
`-override-build-dockerfile` in which case your image will receive advisories that make
it undeployable in any environment.
