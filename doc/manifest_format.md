# The Sous Manifest Format

When you `sous init`,
Sous generates a default manifest for your project.
You can review this manifest by running `sous manifest get`.
If you capture that output
e.g. `sous manifest get > /tmp/myproject.yml`,
you can edit the contents of the manifest and replace it by
`sous manifest set < /tmp/myproject.yml`.

What follows is a review of the the format of
the manifest YAML document format,
so that it's values will be sensible.

```yaml
# Source is the location of the source code for this piece of software.
# It will be set by `sous init` and shouldn't be changed.
Source: github.com/myorg/myproject
# Flavor is an optional string, used to allow a single SourceLocation
# to have multiple deployments defined per cluster.
# It is valid (and very common) to omit Flavor entirely.
Flavor: "vanilla"
# Owners is a list of emails of the owners of this project.
Owners: [ "me@example.com" ]
# Kind is the kind of software that the project represents.
# For the time being, "http-service" is the only useful value.
Kind: "http-service"
# Deployments is a map of cluster names to DeploymentSpecs
Deployments:
  ci-example:
    # Version is a semantic version of the project.
    # To deploy successfully, the version should be built and available in
    # a known-to-Sous docker repo
    Version: "1.2.3"

    # Resources represents the resources each instance of this software
    # will be given by the execution environment.
    # It is a map whose keys are determined by Sous's configuration,
    # but generally conform to this pattern:
    Resources:
      cpus: "0.1" #in units of 'a whole processor'
      memory: "100" #in MB - triggers an OS-level OOM if exceeded.
      ports: "1" #How many network ports to allocate.

    # Metadata stores values about deployments for outside applications to use
    # Appropriate values are beyond the scope of this guide.
    Metadata: {}

    # Env is a list of environment variables to set for each instance of
    # of this deployment.
    Env:
      IS_CI: yes

    # NumInstances is a guide to the number of instances that should be
    # deployed in this cluster
    NumInstances: 2

    # Volumes lists the volume mappings for this deploy
    # Generally speaking, mapping volumes breaks the stateless principle of
    # containerized microservices and they are therefore discouraged.
    Volumes: []

    # Startup contains startup healthcheck options for this deploy.
    # (note that ongoing service monitoring is outside of the scope of the manifest)
    Startup:
      # Services that don't respond to a health check can set this true to skip
      # the whole process and be considered healthy as soon as they're running.
      SkipCheck: false

      # Singularity has a two-phase healthcheck: first it attempts to make a
      # TCP connection to the service. Once a connection has succeeded, then it
      # attempts a HTTP check.

      # These are the configuration options for the TCP connection:

      # The initial delay to wait before attemping TCP connections.
      ConnectDelay:     10 # Singularity:  Healthcheck.StartupDelaySeconds

      # The overall time from first attempt to connect until the service must
      # have accepted a TCP connection.
      Timeout: 30 # Singularity:  Healthcheck.StartupTimeoutSeconds

      # How long to wait between connection attempts.
      ConnectInterval: 1 # Singularity:  Healthcheck.StartupIntervalSeconds


      # Options related to the HTTP transaction check once TCP is established:

      # The protocol to connect over. Must be HTTP or HTTPS
      CheckReadyProtocol: HTTP # Singularity:  Healthcheck.Protocol

      # The path to issue healthcheck polling against during startup.
      CheckReadyURIPath: /health # Singularity:  Healthcheck.URI

      # The port index of the service to connect to (e.g. PORT0 etc)
      CheckReadyPortIndex: 0 # Singularity:  Healthcheck.PortIndex

      # Optional list of early-exit failure status codes - if the response is
      # ever any of these codes, the service will be considered unhealthy and
      # killed.
      CheckReadyFailureStatuses: [500, 503] # Singularity:  Healthcheck.FailureStatusCodes

      # Timeout on each http request during the healthcheck.
      CheckReadyURITimeout: 5 # Singularity:  Healthcheck.ResponseTimeoutSeconds

      # The time between checks.
      CheckReadyInterval: 1 # Singularity:  Healthcheck.IntervalSeconds

      # The number of checks to attempt before giving up and considering the service unhealthy.
      CheckReadyRetries: 120 # Singularity:  Healthcheck.MaxRetries
```

Note that, with regard to healthchecks, Singularity is somewhat inconsistent:
during the initial connection testing, there's a connection interval and an
overall timeout, but the HTTP checks have an interval and a number of retries.
