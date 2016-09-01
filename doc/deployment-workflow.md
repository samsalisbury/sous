# Deployment Workflow

This document outlines how deployments are expected to work in Sous.

## TL;DR

```sh
# set some variables - not required, but makes example clearer
export repo=github.com/opentable/myproject
export base_version=0.1.0
export cluster_name="my-cluster"
export version="$base_version-ci$(date +%s)"

# actual sous commands:
sous build -tag "$version" .
sous update -cluster "$cluster_name" -tag "$version"
sous rectify -repo "$repo"
```

## Background

Sous allows developers to create deployments by tagging source code.
This model of deployment (**source code** -> **running software**) makes the build artifact
(a container or binary)
simply a product of the source code.
The running software is a product of the build artifact plus configuration,
which is stored externally from the source code.

Other models of software deployment treat the build artifact as the primary deployable
(**build artifact** -> **running software**).
This kind of model implies a disconnect between source code and the build artifact,
and can encourage complex build processes, often requiring complex manual setup.

The Sous model of deployment is philosophically designed to promote source code that includes all necessary information about how it is built.
Rather than reading a README, configuring your build machine accordingly, installing dependencies manually, then running a build,
Sous encourages stack-centric conventions, explicit dependency declaration, and automated configuration.
You should be able to build any Sous-compatible project with a single command.

## The workflow from a developer's perspective

0. Register your application with Sous Server using Sous CLI to generate a configuration stub, which you can customise.
1. Push a semver-compatible tag to a commit in your source code repository.
2. _Sous goes to work [building and testing the commit]_
3. You are notified whether the code you tagged produced a valid deployment artifact.
4. If you still want to deploy, send a message to the Sous Server to deploy that artifact.
5. _Sous [verifies the deployment artifact and updates the manifest]_
6. _Sous [rectifies the manifest]_

[building and testing the commit]: #sous-builds-and-tests-the-commit
[verifies the deployment artifact and updates the manifest]: #sous-verifies-the-image-and-updates-the-manifest
[rectifies the manifest]: #sous-rectifies-the-manifest

## Sous builds and tests the commit

1. Sous sees this tag, and interprets that as an intention to deploy.
2. Sous builds a container image based on the source code at that tag.
3. Sous runs automated tests and contracts against the built container image.
4. If the container image successfully passes these tests, the image is stamped with a signed label, verifying that it has passed these tests.
5. The image is pre-cached in all expected target clusters.
6. Sous sends a "ready to deploy" notification. (If step 4 failed, the user would already be notified of the failure.)

## Sous verifies the image and updates the manifest

1. Sous checks that the image being requested to deploy actually exists.
2. Sous checks the image has the necessary signed labels, meaning it is a verified working build.
3. Sous writes to the global deployment manifest that this version should be deployed in the appropriate clusters.

## Sous rectifies the manifest

Sous will constantly perform a task known as "rectifying the manifest".
This is where it takes a snapshot of all current deployment states of all running applications in the clusters it knows about.
It then compares this snapshot with its own manifest declarations, and generates a diff.

Based on this diff, Sous performs whatever actions are necessary to make the "real world" (i.e. running software on known clusters)
look like the intended deploy manifests. This will typically consist of API calls to the cluster scheduler, e.g. Singularity.

So, new updates to the manifest will result in new diffs being detected by the rectifier, thus deploying the software.
For new deploys, or failure states, Sous will send messages to the appropriate place, be that your organisations'
existing monitoring and alerting platform, custom API calls, or simple emails is up to you to configure.
