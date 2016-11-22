## sous planned features

- Multi-datacentre deployment orchestration (coming very soon)
- Declarative YAML-based DSL to define deployments (coming very soon)
- Safely deploy source code to production Global event log HTTP API
  to interrogate, and instigate changes to global state Run projects
  locally in a simulated production environment
- Runs on Mac and Linux (Windows support not currently planned) -
  Use the same tool for local development and in your CI pipeline
- Easily distribute shared configuration using the built-in sous
  server
- Automatically adds rich metadata to your Docker images - Run
  executable contracts against any Docker image, to ensure it behaves
  appropriately for your platform.
- Define platform contracts in terms of
  application interactions
- Automatically build NodeJS and Go code using
  a multi-stage build process that eliminates build-time dependencies from
  your production containers. (Java, C#, Ruby, and other languages coming
  soon.)

## ethos

Sous is designed to work with existing projects, using data they already
contain to determine how to properly build Docker images. It is designed
to ease migrating existing projects onto The Mesos Platform, using
sensible defaults and stack-centric conventions. It is also designed
with operations in mind, tagging Docker images with lots of metadata to
ease discovery and clean-up of images.

Sous works on your local dev machine, and on CI servers like TeamCity,
Jenkins, etc., in the same way, so you can be sure whatever works
locally will also work in CI.
