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
 var project1 = SourceLocation{Repo: "github.com/user/project"}
+var project2 = SourceLocation{Repo: "github.com/user/scheduled"}
 var cluster1 = &Cluster{
@@ -231,6 +249,19 @@ var expectedDeployments = NewDeployments(
                        NumInstances: 5,
                },
        },
+       &Deployment{
+               SourceID:    project2.SourceID(semv.MustParse("0.2.4")),
+               ClusterName: "cluster-2",
+               Cluster:     cluster2,
+               Kind:        ManifestKindScheduled,
+               Schedule:    "* */2 * * *",
+               DeployConfig: DeployConfig{
+                       Resources: Resources{
+                               "cpus": "0.4",
+                               "mem":  "256",
+                       },
+               },
+       },
 )
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

Add a new Manifest to `makeTestManifests` to make up the numbers -
you can either leave an empty one or try to sketch out what you'd expect:
```diff
--- a/lib/map_state_to_deployments_test.go
+++ b/lib/map_state_to_deployments_test.go
@@ -135,6 +137,22 @@ func makeTestManifests() Manifests {
                                },
                        },
                },
+               &Manifest{
+                       Source: project2,
+                       Kind:   ManifestKindScheduled,
+                       Deployments: DeploySpecs{
+                               "cluster-1": {
+                                       Version:  semv.MustParse("0.2.4"),
+                                       Schedule: "* */2 * * *",
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

(The `fillstruct` tool (available in vim-go as `:GoFillStruct` is invaluable for this.)

Now if you run tests, you'll get errors that'll drive fixes to `map_state_to_deployments.go`,
thanks to the Diff method working.
```shell
⮀ go test ./lib -run Bounce
--- FAIL: TestState_DeploymentsBounce (0.00s)
        map_state_to_deployments_test.go:537:

                got:
                {
                // ... snip ...
                  "Kind": "scheduled",
                  "Schedule": ""
                }
                differences:
                schedule; this: "", other: "* */2 * * *"
                env; this: map[CLUSTER_LONG_NAME:Cluster Two]; other: map[]
FAIL
FAIL    github.com/opentable/sous/lib   0.013s
```

In this the changes were like:
```diff
--- a/lib/deploy_config.go
+++ b/lib/deploy_config.go
@@ -35,6 +35,8 @@ type (
                Volumes Volumes
                // Startup containts healthcheck options for this deploy.
                Startup Startup `yaml:",omitempty"`
+               // Schedule is a cronjob-format schedule for jobs.
+               Schedule string
        }

        // A DeployConfigs is a map from cluster name to DeployConfig
@@ -171,6 +173,7 @@ func (dc DeployConfig) Clone() (c DeployConfig) {
        }
        c.Volumes = dc.Volumes.Clone()
        c.Startup = dc.Startup
+       c.Schedule = dc.Schedule

        return
 }
@@ -242,6 +245,12 @@ func flattenDeployConfigs(dcs []DeployConfig) DeployConfig {
                }
        }
        for _, c := range dcs {
+               if c.Schedule != "" {
+                       dc.Schedule = c.Schedule
+                       break
+               }
+       }
+       for _, c := range dcs {
```

And now:
```shell
⮀ go test ./lib -run Bounce
ok      github.com/opentable/sous/lib   0.013s
```

## From Sous Deployments to Singularity objects

Likewise, we're going to add a test that updating the Deployment
is reported as requiring an update to the Singularity Request.

```diff
--- a/ext/singularity/deployer_test.go
+++ b/ext/singularity/deployer_test.go
@@ -329,10 +329,39 @@ func TestScaling(t *testing.T) {
+func TestScheduling(t *testing.T) {
+       startDep := baseDeployment()
+       startDep.Kind = sous.ManifestKindScheduled
+       startDep.Schedule = "* 3 * * *"
+       pair := matchedPair(t, startDep)

+       assert.Equal(t, pair.Prior.Schedule, pair.Post.Schedule)
+       assert.Equal(t, pair.Prior.Kind, pair.Post.Kind)
+
+       pair.Prior.Schedule = "* 2 * * *"
+
+       diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
+       assert.True(t, diff)
+       assert.NotEmpty(t, diffs)
+
+       assert.True(t, changesReq(pair), "Updating schedule reported as not changing Request!")
+       assert.False(t, changesDep(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Deploy!")
+}
+
+func TestSchedulingOnlyForScheduled(t *testing.T) {
+       startDep := baseDeployment()
+       startDep.Schedule = "* 3 * * *"
+       pair := matchedPair(t, startDep)
+       pair.Prior.Schedule = "* 2 * * *"
+
+       diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
+       assert.False(t, diff)
+       assert.Empty(t, diffs)
+
+       assert.False(t, changesReq(pair), "Changed schedule data for HTTP service treated as changing Request!")
+       assert.False(t, changesDep(pair), "Changed schedule data for HTTP service treated as changing Deploy!")
+}
+
```

```shell
⮀ go test ./ext/singularity
--- FAIL: TestScheduling (0.00s)
        Error Trace:    deployer_test.go:342
        Error:          Not equal:
                        expected: "* 3 * * *"
                        received: ""
        Error Trace:    deployer_test.go:347
        Error:          Should be true
        Messages:       Updating schedule reported as not changing Request!
FAIL
FAIL    github.com/opentable/sous/ext/singularity       0.072s
```

Key here is that `matchedPair` produces
what should be a matched `DeployablePair`
by round-tripping the `baseDeployment` into Singularity JSON,
and then back.
This means that it tests all the machinery for converting
Deployments into Singularity Requests and Deploys.

The test that changesReq() is true is contextual to the field we're adding -
it's important that Sous knows then it should send commands to Singularity.
`changesReq` is the easier to fix:
```diff
--- a/ext/singularity/deployer.go
+++ b/ext/singularity/deployer.go
@@ -263,7 +263,9 @@ func (r *deployer) RectifySingleModification(pair *sous.DeployablePair) (err err
 // could report ("deploy required because of %v", diffs)

 func changesReq(pair *sous.DeployablePair) bool {
-       return pair.Prior.NumInstances != pair.Post.NumInstances
+       return (pair.Prior.Kind == sous.ManifestKindScheduled && pair.Prior.Schedule != pair.Post.Schedule) ||
+               pair.Prior.Kind != pair.Post.Kind ||
+               pair.Prior.NumInstances != pair.Post.NumInstances
 }
```

To get the round trip working,
we need to ensure that the schedule makes it
into the generated Request:

```diff
--- a/ext/singularity/deployment_builder.go
+++ b/ext/singularity/deployment_builder.go
@@ -5,6 +5,7 @@ import (
        "fmt"
        "strings"

+       "github.com/davecgh/go-spew/spew"
        "github.com/opentable/go-singularity/dtos"
        "github.com/opentable/sous/ext/docker"
        "github.com/opentable/sous/lib"
@@ -131,6 +132,7 @@ func (db *deploymentBuilder) completeConstruction() error {
                wrapError(db.restoreFromMetadata, "Could not determine cluster name based on SingularityDeploy Metadata."),
                wrapError(db.unpackDeployConfig, "Could not convert data from a SingularityDeploy to a sous.Deployment."),
                wrapError(db.determineManifestKind, "Could not determine SingularityRequestType."),
+               wrapError(db.extractSchedule, "Could not determine Singularity schedule."),
        )
 }

@@ -438,3 +440,15 @@ func (db *deploymentBuilder) determineManifestKind() error {
        }
        return nil
 }
+
+func (db *deploymentBuilder) extractSchedule() error {
+       spew.Dump(db.Target.Kind)
+       if db.Target.Kind == sous.ManifestKindScheduled {
+               if db.request == nil {
+                       return fmt.Errorf("Request is nil!")
+               }
+               spew.Dump(db.request.Schedule)
+               db.Target.DeployConfig.Schedule = db.request.Schedule
+       }
+       return nil
+}
diff --git a/ext/singularity/recti-agent.go b/ext/singularity/recti-agent.go
index c086ca40..9f06007d 100644
--- a/ext/singularity/recti-agent.go
+++ b/ext/singularity/recti-agent.go
@@ -206,12 +206,22 @@ func singRequestFromDeployment(dep *sous.Deployment, reqID string) (string, *dto
        if err != nil {
                return "", nil, err
        }
-       req, err := swaggering.LoadMap(&dtos.SingularityRequest{}, dtoMap{
+       reqFields := dtoMap{
                "Id":          reqID,
                "RequestType": reqType,
                "Instances":   int32(instanceCount),
                "Owners":      swaggering.StringList(owners.Slice()),
-       })
+       }
+       if reqType == dtos.SingularityRequestRequestTypeSCHEDULED {
+               reqFields["Schedule"] = dep.Schedule
+
+               // until and unless someone asks
+               reqFields["ScheduleType"] = dtos.SingularityRequestScheduleTypeCRON
+
+               // also present but not addressed:
+               // taskExecutionTimeLimitMillis
+       }
+       req, err := swaggering.LoadMap(&dtos.SingularityRequest{}, reqFields)
```
