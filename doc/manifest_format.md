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
    # Startup contains healthcheck options for this deploy.
    Startup:
      # The path to issue healthcheck polling against.
      CheckReadyURIPath: "/health"
      # The per-check timeout.
      CheckReadyURITimeout: 5
      # The overall timeout before the service should be considered unhealthy.
      Timeout: 60
```
