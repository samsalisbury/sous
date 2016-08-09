# Bridge Networking

Sous assumes the use of bridge networking in Mesos/Singularity/Docker.
However, it does not expose the portMapping interface.

The reasoning here is fairly simple:
Sous is designed to orchestrate microservices.
It assumes that part of that orchestration will be a discovery service of some kind.
Services will need to register themselves,
and to do that will need to know their actual networking information.
Therefore, configuring port mappings doesn't simplify service code
but it does split up the concern of network configuration into two pieces.

## Howto

The bare components of managing networking under Sous are these:

You need to **bind** to `0.0.0.0:$PORT0` (or $PORT1 or ...) and
**announce** on `$TASK_HOST:$PORT0`.

Binding on 0.0.0.0 is safe and sensible because bridge networking
means that your service is on an isolated network.

For clarity's sake, in the above
`$PORT0` and `$TASK_HOST` are environment variables.
You should use whatever facility your language provides for accessing them.
For instance, in Ruby, it's
`ENV["PORT0"]`.

## Specific Cases

### Nginx

Ngninx is notable for providing facility neither for
specifying ports on the command line nor
for dynamically in its configuration.

It's common practice therefore to
build a template configuration and
pipe it through `sed` to substitute in environment variables on boot.
