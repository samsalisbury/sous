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
The server for `clusterB` has local data about
`service-foo` in `clusterA`,
that that data is considered
merely advisory.

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
it performs conditional
`GETs /state/deployments`
to the respective authoritative servers.
It bases the `If-None-Match` headers on its own local state.
If the GETs come back with 200 instead of 304,
the coordinating node updates its local database.
In the event of a network or HTTP error,
the coordinating server replies with its local most-recent information.

When a coordinating node receives a
`PUT /gdm`
it determines the implied changes,
and partitions the deployments based on cluster.
Those changes are then submitted
to the managing Sous servers via
`PATCH /state/deployments`.
Again, this is a conditional request:
the `If-Match` is based on the local data
of the coordinating server.
Once those responses have come back with 204,
the "coordinating" server records to its local DB,
and returns 200 to the client.

The 204 reponse is used in the "happy path" case:
where the authority's state matches the `If-Match` condition,
and the contents of the request are successfully applied.

The authoritative server may determine that its local state
doesn't match the condition of the `PATCH`,
but that applying the PATCH would have no effect -
in other words, the data contained in the request are already
true about the DeployIDs listed in the request.
In this case, the authority should return
a 200 with its new state, as if a
`GET /state/deployments` has been issued.
The coordinating node,
in this case,
updates its local database.

In the event of the remote responding `412 Precondition Failed` to the `PATCH`
(because the `If-Match` condition fails,
but the request would have implied a change to the authoritative state.)
the coordinating node makes a new
`GET /state/deployments` from the remote,
updates its local database and determine whether to try again.
Specifically, it computes a difference of the new authoritative state
and the state it knew for that cluster when the `PUT /gdm` began.
If there's an intersection between that difference
and the change it is attempting with the `PATCH`,
the overall `PUT /gdm` returns `412 Precondition Failed`.
Otherwise,
it is safe to retry the `PATCH`,
and the coordinating node proceeds by doing so.

If any of the authoritative servers return an error,
in response to `PATCH /state/deployments`
the coordinating server returns a 503 error
the body of which indicates which authoritative server(s)
were unable to service the request,
which the client relates to the operator,
so that they can potentially adjust and retry their request.

Additionally,
as authoritative servers accept
`PATCH /state/deployments`
requests
(i.e. after having retuned 204 responses),
they issue new
`PUT /advisory/gdm?cluster=$name`
requests
to all known siblings
(including their own cluster name
in the query parameter.)

Each server receiving this `PUT` request
uses the request body to update its database
about Deployments for which it is *not* authoritative.

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
we get consistency and progress.

In terms of consistent reporting,
first, the advisory reporting on updates
communicates values to non-authoritative servers when those values update.
The second is the live update,
and fallback to cached values.
This serves as the definitive update mechanism,
in a kind of "Just In Time" pattern.

One general problem with distributed agreement
has to do  with simultaneous updates.
That is, if two clients
(sending messages to the same or different coordinating servers)
send conflicting data.
This is the motivation behind all of
the PUTs and PATCHes laid out above being conditional
(i.e.  with `If-Not-Modified` headers attached)
and the GET-and-retry behavior
when `PATCH` returns 412.
