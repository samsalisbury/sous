# sous features

- Automation
  - Deploy straight from code to production.
  - Deploy globally as standard
  - Spin up other teams' applications with one command.
  - Run platform contracts in CI
  - Run your own smoke tests in production.
- Consistency
  - Single, unified deployment workflow.
  - Easy to deploy any team's application.
  - Centralised environmental configuration.
  - DRY configuration: specify only the things that change.
  - It's easier to open-source code built with Sous.
  - Application == behaviour, environment == configuration.
- Lucidity
  - Declarative deployments are self-describing.
  - Centralised knowledge of global deployments.
  - Always know which version of each service is deployed in each environment.
  - No need to pollute your code with deployment-specific information.
  - Clear, detailed logs. You can manually play back anything sous does.
- Speed
  - Spend no time configuring TeamCity
  - Spend no time wondering how a service is built/configured/deployed.
  - Spend no time setting up multi-datacentre deployments.

- Build
  - Local developer build
  - Operational builds
- Deployment
  - Global deployment state
  - Discovery and update of deployment to reflect declared global state

# feature comparison

This is a work in progress...

| Feature                              | Spinnaker | PaaSTA | Nomad | Swarm | Compose | Otto | Sous |
| ---                                  | ---       | ---    | ---   | ---   | ---     | ---  | ---  |
| Deploy straight from code.           | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
| Global deployments as standard.      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
| Easily launch any app locally.       | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
| Run universal platform contracts.    | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
| Run custom smoke tests in all envs.  | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
| Single, unified deployment workflow. | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
| Easily deploy any application.       | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
| Centralised environment config.      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
|                                      | ?         | YES    | ?     | ?     | ?       | ?    | YES  |
