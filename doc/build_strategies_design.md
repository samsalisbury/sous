# Build Strategies

*This document will be an explanation of Build Strategies and how they work.
For the time being,
it documents our design process and ideas.*

## Design Goals

sous build should be able to

* Build based on a named configuration
  (which we expect to be a build docker image name).
* Cache fetched dependencies
  (e.g. Maven's .m2 or node's node_modules or Ruby gems).
* Cache intermediate build products
  (for local development.)
* Produce small-as-possible product image.
* Produce more than one product image.
* Users should be able to make small overrides to the build process,
  if only so that quick experiments are possible
* Sous controllers should be able to audit the build processes in place,
  so that they can be consolidated -
  i.e. "small overrides" should be reasonably public so that we can find audit divergences.
* Sous init, or a related tool,
  should be able to guess at the "best" named build configuration
  and set it up automatically, or provide options to the local operator
  about what to choose.
  `sous build` should hint at this tool
  when a configuration is unavailable or sub-optimal
  (i.e. at some point the "simple dockerfile" strategy should suggest there might be a better way.)


## Current Proposal

The manifest for the project has a sub-entry like:
```
  Build:
    Type: mount-run-split
    Image: docker.internal.com/our-maven:latest
```

`mount-run-split` is the goal design here.
We might bless `simple-dockerfile` and `build-copy-split` as well.
For the time being, the type is superfluous - if there's
a non-empty image field, we proceed with this proposal.
It provides an avenue for future alternatives,
however.

We start by synthesizing a small Dockerfile like
```
FROM docker.internal.com/our-maven:latest
```
`sous build` "builds" this container.
(We could simply `docker pull` the image,
but we get the advantage
of having a single code path for one-off experiments.)
It then `docker run`s it with three mounted volumes:
one for fetched dependences,
one for intermediate products
and one for output products (i.e. jar files).
The runspec is delivered into the output volume
by the build tools and scripts in the build image,
and Sous uses the runspec to scatter products in the output into run images.
The actual build images will also be responsible
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
to the final deployed images,
which would give us the data required for audits -
effectively, we could chase the long tail of the histogram
of "build image digest"
(because variant builds would have a different digest -
I'm 85% sure that "conformant" builds would all have the same digest),
and other labels would provide enough data
to track down the actual built dockerfile.

We could add a /sous directory to built deploy artifacts
and store e.g. the build Dockerfile there for reference purposes.

In order to allow for easy development of new build containers,
and to effect transition to existing ones,
`sous build` would be extended with two new flags:
`-dev` would allow a number of non-production-suitale changes
to the Sous build process, like
caching of intermediate artifacts,
flags to build programs to output more logging.
`-override-build-image <Dockerfile>`
would only be available in `-dev` mode,
and would use the named Dockerfile
instead of consulting the manifest.

Right after getting the manifest to build from,
Sous should issue `git config` commands to store
the type and
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
even if it's to have `Type: simple-dockerfile`.
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
