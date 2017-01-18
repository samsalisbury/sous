# Sous Drives Builds

Sous's design includes the idea of repeatable builds.
As a design goal,
this means that a service built and deployed by Sous
can be built in any environment that satisfies *Sous's* requirements.
(That is: as opposed to the application's build chain's requirements.)
Furthermore, this means that Sous can certify that
the resulting service artifact
is determined completely by
the source code that Sous knows about
and records into the metadata of the resulting image.

This design goal
is in service to the feature requirement
that Sous be able to provide a
trustworthy, accountable
deployment architecture.

Sous accomplishes this by providing a single command
(`sous build`)
which performs a flexible but predetermined build process.
Under the covers, `sous build`
effectively runs `docker build`.
The build process is
determined by a Dockerfile
in the project source repository.

## More Pleasant Than True

Unfortunately,
the present implementation of Sous takes
the resulting Docker container
and deploys that.
One consequence is that
the build toolchain for each service
is deployed with the executable.
This is undesirable for a number of reasons:
the resulting container is larger than necessary,
and provides a larger attack surface for the container.

A significant planned feature of Sous
(currently called "buildpacks,"
but that term is somewhat overloaded)
would use a project local Dockerfile
to build artifacts,
and then copy those artifacts into
an tightly constrained "runpack" Dockerfile,
templated by Sous.
This would accomplish the
repeatable, certified builds
without carrying the toolchain along into production.

The advantages of these builds
will be several.
Teams and projects would
be able to share whole toolchains very easily.
Cross-platform builds will be supported as standard,=
by ensuring a consistent Dockerised build environment.
Certain paper processes
surrounding the deploy process
could be replaced outright
with an auditable automatic system.

## For the Moment

Sous already captures
a series of "QA advisories"
which enumerate all of the ways
in which a particular build can diverge
from Sous's ideal clean build.
These include builds in workspaces that aren't completely checked in,
or where the current commit isn't pushed,
or isn't tagged with a Sous-deployable version tag.

For the time being,
the recommended deployment configuration
for Sous servers
will cheerfully accept containers
with advisories marked onto them.
This is in deference to the above mentioned
lacks in the buildpack system.
As those are corrected,
those configurations will be changed,
and services will need to conform to
"clean build" practices.

## What You Should Do

In the meantime,
teams have two basic choices
(which can, of course,
be made on a per-project basis.)
First, jump in on the "pure" build approach.
Push all your build chain into your Dockerfile,
so that you copy your source code in,
and use your build products as
your Docker "CMD."
Second, you can maintain a current build process,
possibly changing whatever steps build an actual container
with `sous build`.
Since Sous is designed as the driver of the build process,
the latter approach is refered to as the "inverted" workflow.

The advantage of using the "proper" Sous approach is that
you'll start constructing the "buildpack" Dockerfile
that your service will need when we convert
to buildpacks.
Your overall toolchain will be usable almost unchanged
when the time comes.
For instance, if you use TeamCity with Sous,
your TeamCity configurations will remain pretty much the same.
You project Dockerfile will change slightly,
since you'll deliver your build products to Sous
rather than executing them directly.
Ideally you'll be able to make your updated Dockerfile
into something that you
(or other projects or teams)
can start `FROM` in the future.
The disadvantages are as discussed above:
(in the interim)
a bigger container size,
and a larger attack surface.

The advantage of the "inverted" approach is that
you may well already be doing it,
so you may be able to use Sous with very little change
to your existing workflow.
Many build chains can be converted by changing one line like
`docker build` to
`sous build`.
The disadvantages have to do with work required in the future
to retrofit to the buildpack solution,
as well as missed opportunities to share a build chain.
