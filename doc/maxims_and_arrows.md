# sous maxims and arrows

A collection of musings on specific design decisions we're taking as we build Sous.

- [Configuration should live separately from source code]

[Configuration should live separately from source code]: #configuration-should-live-separately-from-source-code

## Configuration should live separately from source code

One of the design goals of Sous is to decouple the behaviour of an application from the environment it’s deployed in.

This is important in order to avoid the overhead of synchronisation between operations and product engineering teams, and should free up operations to make changes to the infrastructure without requiring every application deployed on that infrastructure to have its source code modified. “Source code” here means “the code in the repository”.

Of course, there are some benefits to having configuration in the same repository as the behaviour of an application, if, for example you want to directly deploy that app without having to refer to any external source. Maven provides nice tooling around separate packaging of configuration, so in this case I can see the attraction. (However, we’re building tools that will make this alternative easy to use for all projects, see below.)

Synchronisation between teams is extremely expensive (think about making changes to load balancers, DNS, alerting configuration etc). So at the moment, we are strongly of the opinion that configuration which changes per environment should be stored alongside that environment, rather than with each application that is deployed there. This has a number of other benefits:

- De-duplication (should every application have a separate record of where the logging/discovery endpoints are for each datacentre?)
- Easier discovery (one place to look to see how an app is configured, not searching for JSON/XML/YAML in a repo, and then reading the code to figure out how it’s loaded/which config is loaded).
- Easier to open-source (configuration is deployment-specific, and not considered clean for open source)

In order to ease the workflow when using external configuration, we are building tools that make it really easy to query and modify configuration, as well as to run any application as if it were in any environment. Config will be cached on your dev machine, so everything works offline, and you don’t need to always query a central source. We hope these features make centralised configuration as painless as possible for day-to-day workflow.
