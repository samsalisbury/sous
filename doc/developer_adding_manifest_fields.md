# Adding Fields to Manifests

As central parts of the Sous model,
the Manifest and Delployment structs
tend to be key to many new features in Sous.
For the same reason,
it's very important
to make sure that
those changes are made in the right way.

## Add to Deployment

Make you first change to the Deployment struct
(or one of it's embedded structs)
in `lib/deployment.go`.
```diff
--- a/lib/deployment.go
+++ b/lib/deployment.go
@@ -38,6 +38,8 @@ type (
                Owners OwnerSet
                // Kind is the kind of software that SourceRepo represents.
                Kind ManifestKind
+               // Schedule is a cronjob-format schedule for jobs.
+               Schedule string
        }
```
Basically, just add the field you need.
If you run tests
Tip: you can focus like:
```shell
⮀ go test ./lib -run TestDeploymentDiffAnalysis
--- FAIL: TestDeploymentDiffAnalysis (0.03s)
        Error Trace:    deployment_test.go:50
        Error:          Should be empty, but was [Deployment.Schedule]
FAIL
FAIL    github.com/opentable/sous/lib   0.032s
```
The failure will look like that.
It means that
the `Deployment.Diff` method doesn't consider your field.
(in this case Deployment.Schedule.)

## Fix Diff

Add code to `Deployment.Diff` to consider your field across Deployments.
```diff
--- a/lib/deployment.go
+++ b/lib/deployment.go
@@ -231,6 +231,13 @@ func (d *Deployment) Diff(o *Deployment) (bool, Differences) {
                diff("kind; this: %q; other: %q", d.Kind, o.Kind)
        }

+       // Schedule is only significant for Scheduled Jobs
+       if d.Kind == ManifestKindScheduled {
+               if d.Schedule != o.Schedule {
+                       diff("schedule; this: %q, other: %q", d.Schedule, o.Schedule)
+               }
+       }
+
```

Run the test again:

```shell
⮀ go test ./lib -run TestDeploymentDiffAnalysis
ok      github.com/opentable/sous/lib   0.032s
```

## Ensure that Deployments and Manifests Map

Add to the `map_deployments_to_state_test.go` like:
```diff
--- a/lib/map_state_to_deployments_test.go
+++ b/lib/map_state_to_deployments_test.go
@@ -10,6 +10,7 @@ import (
 )

 var project1 = SourceLocation{Repo: "github.com/user/project"}
+var project2 = SourceLocation{Repo: "github.com/user/scheduled"}
 var cluster1 = &Cluster{
        Name:    "cluster-1",
        Kind:    "singularity",
@@ -135,6 +136,21 @@ func makeTestManifests() Manifests {
                                },
                        },
                },
+               &Manifest{
+                       Source: project2,
+                       Kind:   ManifestKindScheduled,
+                       Deployments: DeploySpecs{
+                               "cluster-1": {
+                                       Version: semv.MustParse("0.2.4"),
+                                       DeployConfig: DeployConfig{
+                                               Resources: Resources{
+                                                       "cpus": "0.4",
+                                                       "mem":  "256",
+                                               },
+                                       },
+                               },
+                       },
+               },
        )
 }
```

This'll cause new test failures.
Some will be assumptions the tests make:
in this case, we've added a new manifest, and
the test Deployments only produce 2.
```shell
⮀ go test ./lib
--- FAIL: TestState_DeploymentsCloned (0.00s)
        map_state_to_deployments_test.go:261: deployments different lengths: expected 4, got 5
--- FAIL: TestState_Deployments (0.00s)
        map_state_to_deployments_test.go:512: deployments different lengths, expected 4 got 5
--- FAIL: TestDeployments_Manifests (0.00s)
        map_state_to_deployments_test.go:530: got 2 manifests; want 3
```




* Deployment <-> Manifest
* SingReq/SingDep <-> Deployment
* SingRep/Dep -> Deployment: deployment_builder
* Deployment -> SingReq/Dep: recti-agent, deployer
    * changesReq & changesDep
