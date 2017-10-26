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

The named configuration is configured using the one-line-dockerfile - which points to a build container.

`sous build` "builds" this container,
and then `docker run`s it with three mounted volumes:
one for fetched dependences,
one for intermediate products
and one for output products (i.e. jar files).
The runspec is delivered into the output volume,
and Sous uses the runspec to scatter products in the output into run images.
The actual build images will be responsible
for arranging the volumes into their needed directory structure
(e.g. symlinks or mount -o bind)
However, part of the recognition process
for "mount-run-split" images will be enviroment variables specifying those three mount points.
(The output mount will be required, the caches will be optional.)

Because the build image will generally just be downloaded,
the `docker build` of the initial container will generally be (almost) a no-op -
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
Alternatively/additionally,
we could add a /sous directory to built deploy artifacts
and store e.g. the build Dockerfile there for reference purposes.

At some point in the future,
we can convert to a notarized platform
where the build images would need to be notarized
in order to be used
(or at least "unsigned build image" becomes an advisory)
which would encourage/require
that updates to the build images be reviewed and approved.

## Discussion

**SS:** After spending some time digesting this I generally agree with it. Some
comments below which I would like to discuss, as this proposal/current
implementation means adding sous-specific code (Dockerfiles) to the users’
repos which presents a bit more complexity for querying/auditing than should be
necessary.

**JL:** I think we've got two reaonable options for configuring a strategy selection:
*some* in-repository file,
or a manifest level GDM field.
The problem I have with putting this exclusively in the GDM
is that it makes the idea of "small overrides" impossible.
If it's possible to put things in a local file,
then we lose a lot of the advantage of centralizing to the GDM -
auditors still need to investigate the repo
to confirm there isn't an override file.

Elsewhere we've discussed the idea of `sous build -dev` -
perhaps in -dev mode, you can specify `-override-build-strategy <file>`
for the purpose of quick iteration.
It doesn mean that no "special snowflake" application can exist -
anything built in production has to have a blessed strategy to work from.

Eventually, I'd want to add a command to fetch the dockerfile for
a named strategy,
so that a user can edit it.

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

**SS:** (the strategy dockerfile should be) Named anything but “Dockerfile” I
think, since “Dockerfile” is generally expected to produce a runnable image on
‘docker build’. This would mean users can still have a normal Dockerfile for
other purposes (even non-sous purposes).

**JL:** I think that's reasonable, especially in the context of the forgoing:
a `-dev` mode override.

**SS:** The use of this build Dockerfile in question also means de facto we have
Sous-specific stuff in the repo. I’m not super delighted about this, but I can
see it solves the problem of being able to run builds without access to the
GDM. My preference would be that the named builder be stored in the GDM, and
that the entire GDM be cached locally on each lookup, so builds can still
proceed as long as there is a recent GDM cached locally. ‘Sous build’ should
have a flag to pick a different builder, which would then be recorded in the
GDM when the image is recorded. One of the benefits of this is that local
tooling would not need to write any files to the user's repo to record
sous-level decisions, and having that decision in the GDM should make it easier
to query.

**JL:** Perhaps Sous could add a *git* config to the current repo like
`sous.build-image-name` which it could fall back to if accessing the manifest
fails.

That addresses the issue of momentary access problems
(the air-travel use case)
but not fresh builds.
Also to note: I've more than once been challenged
on the significance of recording *nothing* sous specific in the project itself.
One observation is that a config file hints at how the project is intended to be built.
That is: if you find sous.config in the root of a project,
you'll at least ask in chat what that file is for.
If there's nothing Sous specific, it might not occur to you.

**SS:** This is a definite benefit of using a Dockerfile.  Maybe (since its use
should be exceptional), we could still allow this even under the proposed
scheme above where named builder is recorded in the GDM? E.g.
Dockerfile.sousbuildoverride which if present would completely override what
the GDM says, and allow for rapid development. The presence of these files
would likewise be easy to audit to find common patterns that should themselves
be named builders.

**JL:** see above re `sous build -dev -override-strategy <file>` -
I prefer that to the implicit override of the file in the repo
which e.g. might be committed and forgotten about.
