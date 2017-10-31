# Build Strategies

*This document will be an explanation of Build Strategies and how they work.
For the time being,
it documents our design process and ideas.*

## Design Goals

`sous build` should:

* Build using a Docker image
  (known as a "builder image")
  configured in the manifest.
* Produce runnable Docker images as output (known as "product images")
* Cache fetched dependencies when building on developer machines
  (e.g. Maven's `.m2` or node's `node_modules` or Ruby gems).
* Cache intermediate build products when building on developer machines
  (for rapid local development.)
* Perform a clean build at will
  (for continuous delivery).
* Produce the smallest possible product image layer (i.e. the part that represents
  the application being built rather than its runtime dependencies).
* Produce more than one product image per source code repository.
* Users should be able to make small overrides to the build process
  locally on their machine
  to facilitate experimentation.
* It should be possible to audit the build images in use
  so that they can be consolidated.
* `sous init`, or a related tool,
  should be able to guess at the "best" named build configuration
  and set it up automatically, or provide hints to the local operator
  about what to choose.
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

* `mount-run-split` meaning we run the `Image` with various directories from
  the local machine mounted. The result is that one of those directories is
  populated with built runnable artifacts. We then split those artifacts
  amongst various output containers ready for deployment.
* `simple-dockerfile` meaning we simply build the `Dockerfile` in the project's
  source code repository (actually `SourceLocation`). This `Strategy` would not
  use the `Image` field at all and would is capable of producing only one
  artifact at a time (the docker image produced by `docker build`).

Simple-dockerfile strategy builds are already supported by Sous, and are currently the
default when a Dockerfile is present in the working directory when executing
'sous build'.

The rest of this document describes in more detail the `mount-run-split` build
strategy, which we think should be the standard strategy used by most projects.

### mount-run-split builds

We start by synthesizing a one-line Dockerfile in memory
based on the image named in `Build.Image` from the YAML proposed above.

```
FROM docker.internal.com/our-maven:latest
```

`sous build` builds this image, known as the "builder image", using `docker build`.
We call it the "builder image" because it
is used to build the project, and is not itself a deployable artifact.
(We could simply `docker pull` the image,
but we get the advantage
of having a single code path for one-off experiments, see below.)

We then `docker run` the built image with three mounted volumes:

* /external for externally fetched dependencies
* /working for intermediate products
* /output for output products (e.g. jar files, directories, executables etc.)

Each builder image compatible with `mount-run-split` must contain a `runspec.json`
file (typically at the root of the filesystem, although this can be overridden by
adding the line `ENV SOUS_RUN_IMAGE_SPEC={some-other-path}`).` This runspec file
specifies how runnable artifacts produced by running the build container will be
executed once they are delivered into runnable docker images (see below).

The runspec is delivered into the output volume
by the build tools and scripts in the build image,
and Sous uses the runspec to scatter products in the output into run images.
The builder images will also be responsible
for arranging the volumes into the directory structure
required by the build tools
(e.g. symlinks or mount -o bind for ~/.mvn2).

### Observations

Because the build image will generally just be downloaded,
the `docker build` of the initial container will usually be (almost) a no-op -
the first time there'll be download of data,
and thereafter there'll be a check of the local daemon's cache of the build image.
However, using a Dockerfile for this purpose allows for the "small changes" required.

It should be possible for Sous
to add labels related to the build process
to the final build images
for auditing purposes.

We could add a /sous directory to built deploy artifacts
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
and would use the named Dockerfile
instead of consulting the manifest.

Right after getting the manifest to build from,
Sous should issue `git config` commands to store
the strategy and
the image
that it discovers.
Then, if on a future build it can't get the manifest
(e.g. no network connection, Sous server is down)
it can consult the local git configuration
to recover that information.

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

We might have a period where we encourage users to switch
to having a Build section
even if it's to have `Strategy: simple-dockerfile`.
Then, we'd trust what we find in the manifest,
but fall back to looking for a Dockerfile.

Finally, we'd come to a point where we require the `Build` section,
and error out if it's missing.

`sous init` would need to be updated to default to adding a `Build`
before the final state.
Ideally, it should "guess" at an image up front.


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

**JL:** By limiting the manfiest entry to an image, it means that "blessed" build images will need to be built and pushed external to a particular project. (Our internal set of build images being an example of that process.) I don't want to start inlining Dockerfiles into the manifest - that seems like an auditing nightmare.

**SS:** So for anything that wants to be deployed, you must push a build image that
can build it, experimentation is strictly off-line on local dev machines. Any builder
image in your source code repository will be ignored unless using
`-override-build-dockerfile` in which case your image will receive advisories that make
it undeployable in any environment.

