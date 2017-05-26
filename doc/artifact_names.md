# Artifact Naming

Sous takes the opinionated stance that deployed artifacts should be named
based on the primary repository of their source code,
and a semantic version.

## Deploying Artifacts

When Sous deploys an artifact,
it uses a unique name for that artifact.
For instance, Docker images are referred to
by their digest name,
rather than a versioned label.
The reasoning behind this is that
it means that the scheduler is free
for instance
to move the service between execution agents,
and when it restarts,
it'll be exactly the same service as was running before.

At deploy time,
Sous resolves the version in the service's manifest
into a unique name.
In the case of Docker,
we find the appropriate image,
and cache the digest of that image.


## Scenarios Addressed

For design purposes, there are a number of situations
that the Sous naming and deployment algorithms
need to address.

### Restarted Services

If a service was deployed by an operator,
restarts of those services should not lead
to a new artifact being deployed.
In other words,
having deployed a service successfully,
I should never worry about its behavior changing
except as a result of a new Sous deploy.

### Rebuilt Services

If a user builds an artifact
a subsequent time
from the same revision of the source code,
we would like to believe
that the resulting image
would behave precisely the same
as the original image.
We can't be certain of this, however.
Therefore,
the first successful build of a revision
has to be considered the canonical image,
and all deploys of the Sous name for that image
should all represent the first image.

### Ephemeral Tags Becoming Real

Sous allows for builds
with a specified image tag
using a `-tag` option.
When this tag doesn't exist in the source repo,
the tag is considered "ephemeral"
and this is recorded as an advisory on the image.
If that tag is later recorded in the source repo,
a build on the tagged revision should be considered
canonical and used in preference to ephemeral images.

### Builds on the tag versus past the tag

If Sous isn't given a tag
when building,
it defaults to the "most recent" tag.
This is the same logic used by `git describe`.
If there is already a build based
on the revision that the tag points to,
builds "past" the tag
(i.e. the already built tag
is their "most recent" tag)
should not be preferred over
the canonical build that was made on the tag itself.

## QA Advisories

These (and other) scenarios
led to the implementation of
a QA system within Sous builds,
which includes in the labeling on each image
a list of advisories
about how the build was in one way or another
"unhygienic."
Generally speaking, then,
for a particular tag,
the image with the fewest advisories
should be considered "most canonical."
The "rebuilt" advisory means that the absolute case
(i.e. zero advisories)
is unambiguous: it's the
first build
on a revision
that actually has the git tag with the same name.
