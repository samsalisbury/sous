# Cluster and Container Interface

Sous uses an abstraction of a running service called a Source ID.
This incorporates
the official repository for the source code,
and offset within that repository (usually a relative path from the root)
the [version](http://semver.org) of the software
and the revision of the software.
The Source IDs are composed out of the manifests in the GDM
to determine what should be deployed where.

Sous has to be able to record and recover metadata sufficient to construct a source ID
from both the running cluster
and from the container registry.

Sous maintains the state of deployments by comparing
the data in the container registry
and the running clusters.
It therefore must be able to make an accurate comparisons
between what's actually running
and the images available in the registry.

## An implementation

Using Docker and Singularity as an example,
Docker allows for labels on its images.
Sous uses these labels to record the source ID
and build advisory metadata
into the images themselves.

So, given a GDM,
Sous composes the source ID from the manifests.
It retrieves the image names from the clusters
and gathers the labels from the registry
to compute the source IDs that are deployed.
It compares the two,
and where changes need to be made,
it looks up the appropriate images from source IDs
and issues deploy commands to the Singularity clusters.

However,
the Docker Registry doesn't allow for the image labels to be searched on,
only inspected on known image names.
This is a known and designed restriction of the Docker registry -
the premise of that team is that search engines will be built
to suit the demands of various projects.
Sous does exactly that.

Every node where Sous is run collects image names
with useful source IDs.
Key is that these local search caches are built
by making requests of the central registry
and caching the results.
Finally, the Docker constructed content-addressable name
(also known as the "digest name" or "canonical name")
is always collected,
and used in deployments.

By using the content-addressable name
Sous can be sure that the image being
(or already)
deployed is the same as
the image being considered.

The search cache is
first populated by requests made of the images deployed,
then by querying the Docker repos of those images
(i.e. the path to the image previous to the ':' or '@' -
the "sibling" images of the image deployed)
and finally by building an appropriate image.

### On Content Addressability

While meaningful image names were considered, there are a number of drawbacks.
First, changes to design in Sous
would mean changes in the format in the names.
(Note that as of this writing,
the idea of "advisory metadata" is being considered,
which would need to be recorded in the image name somehow.)

Second, Docker does not itself attempt to provide
reproducible builds,
and even if it did,
no platform we know of provides reproducibility
out of the box.
Consequently,
builds performed on separate nodes
(a decided possibility with a globally distributed Sous system)
would appear to be same image,
which would could result in
deployments of a service in some clusters
behaving differently than others, with no indication that they weren't identical.

At root,
this is a data consistency issue,
with software deployment as its scope.
By consistently deploying the precise builds of an application identically,
we can stem off an entire category of bugs before they happen.

### On Statefulness

The search cache adds an layer of state to Sous
that might reasonably be avoided.
It is, however, exactly that:
a cache of data retrieved from more expensive queries against
the central Docker registry.
Since the cache can obviate the need to build new images,
it's important in order for Sous to maintain its guarantees of swift deployment.
