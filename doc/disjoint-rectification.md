# Disjoint Rectification

Until Sous has working per-cluster servers,
it will be important to be able to do rectifications at the command line.

There's a hazard to doing this,
as it relies on local copies of the global manifest,
and changes across those differences will clobber one another during rectification.
To help address this,
`sous rectify` accepts flags to limit the scope it will consider.
It's normal for single teams to use
`sous rectify -repo github.com/team/project`
for instance.
Note well, however,
that if there's a node that runs
`sous rectify -all` without having merged all other manifests
it will remove the deployments made by `-repo` runs.

It is recommended that either
all deployment handled by Sous be done through "disjoint" deploys
(i.e. everyone uses '-repo')
or else
all deployments are done with `-all`,
and modifications are made to a central, reviewed manifest.

Once Sous attains its server feature,
servers will implicitly run in a `-cluster` mode
because each server will be responsible for its local cluster.
At this point,
most of the use cases that drove constrained rectification
will be obsolete, but limited uses are still forseen.
