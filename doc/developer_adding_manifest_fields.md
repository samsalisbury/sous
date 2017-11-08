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



* fix Diff
* Deployment <-> Manifest
* SingReq/SingDep <-> Deployment
* SingRep/Dep -> Deployment: deployment_builder
* Deployment -> SingReq/Dep: recti-agent, deployer
    * changesReq & changesDep
