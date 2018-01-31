# Global Consistency

One of Sous' core requirements
is to maintain a global record of intended deployments
and to ensure that the record is true
with respect to what's actually deployed.

There's a further requirement that,
for instance, if the architecture were to experience a net-split,
engineers local to a particular cluster
should be able to administer their cluster.
When the split is healed,
the global view should return to consistency.

We find that we have what amounts to
a problem of distributed consistency to solve.
Sous needs a system that provides consistency and forward progress.
That is, Sous must:

* report values from the GDM(Global Deploy Manifest) that were issued to it at some point in the past
* report its most recent understood value (not some arbitrary past value)
* recording new values should advance the most recent value as much as possible

## Design

_(the following is currently being implemented)_

The Sous GDM is stored in PostgreSQL databases,
one per cluster,
each local to the managing Sous server.

Each node is authoritative
about its local Deployments.
That is,
whatever version the `clusterA` server has for
`service-foo` in `clusterA`
is the definitive version for
`service-foo` in `clusterA`.

The server for `clusterB` has no data about
`service-foo` in `clusterA` -
instead, if needed,
it queries the `clusterA` server.

Sous clients communicate with a single server -
although which server is a per-request determination.
In the event of network isolation
(or simply by preference)
a user might configure their workstation to talk to their "closest" Sous server.
The server handling a client request
coordinates the response across the Sous infrastructure.

## Protocol Specifics

During a `sous deploy`,
the Sous client updates the Global Deploy Manifest.
(Once the update is complete, the client monitors the resulting deploy
via separate requests to the responsible server nodes.
That part of `sous deploy` is out of scope of this document.)
To do this it starts by issuing a
`GET /gdm` HTTP request
to its preferred Sous server.
(Hereafter, this node is referred to as the "coordinating" server.)
After manipulating the GDM data it receives,
the client sends the new state back via a
`PUT /gdm` request, conditional on "If-Match".

When a coordinating node receives a
`GET /gdm`
it performs
`GETs /state/deployments`
to the respective authoritative servers.
If any authoritative servers fail to respond,
or return an HTTP error status,
an annotation is returned in
the `/gdm` response.

When a coordinating node receives a
`PUT /gdm`
it partitions the deployments based on cluster.
Those changes are then submitted
to the managing Sous servers via
`PUT /state/deployments`.

This is a conditional request:
the `If-Match` is cached from the `GET`
at the coordinating server.
(Just a map of `/gdm` Etags to
records of (cluster,etag) pairs.)
Once those responses have come back with 204,
the "coordinating" server records to its local DB,
and returns 200 to the client.

The 204 reponse is used in the "happy path" case:
where the authority's state matches the `If-Match` condition,
and the contents of the request are successfully applied.

In the event of the remote responding `412 Precondition Failed` to the `PUT`
(because the `If-Match` condition fails,)
the coordinating node
likewise returns `412` to the client,
who retries the update.

## Implementation in Sous

As to actual implementation,
this synchronization is managed with a decorator StateManager.
The StateManager interface has two methods:
ReadState and WriteState.
The GlobalSyncStateManager wraps another StateMananger
(i.e. PostgresStateManager)
and manages the HTTP synchronization requests
(the ones to /state/...  and /advisory/...)
around Read/WriteState calls to the wrapped manager.

## Discussion

When a client issues an update to a value,
the coordinator echoes those values on to the single authoritative server.
At which point,
there are three possibilities:

0. the message never reaches the remote authority
  (it's otherwise not recorded) we rely on *some* error condition to detect this state,
  in the worst case, a network timeout.
0. the message reaches the remote authority and is recorded,
  but the acknowledgement fails - likewise, we receive an error.
0. the message is recorded and we receive the 200 ACK -
  this is the "happy path" and what we should expect most of the time.

We cannot reliably distinguish cases 1 & 2 -
the error messages we receive can be collapsed for purposes of consideration to "no network ACK."
We report this to a human operator as
"This deployment not verified - it may still deploy.
These are the clusters not reporting: ..."

The remote authority, conversely, can't distinguish 2 & 3 if its ACK is lost.
Without devising a global quorum,
we can't avoid that uncertainty.
We may be able to tune it though,
since the practical fact is that there are error cases that do distinguish cases
(i.e.  a complete TCP failure has got to be #1, and TCP makes the likelihood of #2 quite low.)

Regardless,
assuming that the client defaults to some number of retries,
and we accept that the remote authority
receives updates and proceeds with a best effort,
we get consistency.

We don't get theoretical progress because
it's possible for a series of clients
to issue conflicting updates
and retry them indefinitely.
Practically, though,
at some point there's one
"last client standing",
whose update "takes,"
we enter a quiet phase,
and the disappointed clients
retry after that.
This is no worse
than the current situation.

# Failure Modes

The Sous GDM storage design
is intended to address
a number of potential
fault modes.

First, we must consider
the case in which
the network link between a coordinating node
and the Sous authority for a particular cluster
is down (or has enough latency it might as well be.)

Second, the case where
the link between the client
and a particular cluster is down.

Third, where the Sous server
for a particular cluster
has crashed, and is unavailable.

Or worse, where the Sous server
for a particular cluster is misbehaving.

Finally, we have the case where
the network is working well,
and all the Sous servers are running correctly,
but one of the the services upon which Sous depends
is down.
For instance, if the Singularity controller
for a cluster is down,
or the Docker registry.

In these cases,
how do we want Sous to behave?
Our survey of
our engineering organization
leads us to believe that:
* It is better to do a partial update
  than to refuse
  in the case of an absent cluster server.
* The requirement to control
  that partial update
  is at best weak.

The reason for this is effectively that
every service should be able to coexist with
itself in different versions *anyway* -
otherwise rolling updates would be impossible.

What semantics
of deployed artifacts can we accept?
Here, the requirements are stricter.
A deployment command needs to
resolve or fail
in some kind of reasonable time.
(absolutely less than 30 minutes.)
