# Local Development Setup

To contribute to Cluster Registry, you need to set up a local environment.

## Initial setup

Cluster Registry development stack consist of the following applications that runs in Docker containers:

* cluster-registry-api
* cluster-registry-client
* local aws sqs
* local aws dynamoDb
* local k8s cluster created with kind
* local oidc mocking provider

If you want to test one of the Cluster Registry components call directly setup script by specifying to/not to run that component:

* run the stack without cluster registry api: `CLIENT=true make setup`
* run the stack without cluster registry client: `API=true make setup`
* run the stack with cluster registry api and client: `make setup` or `API=true CLIENT=true make setup`

To clean up your local setup run:
`make clean`

Open your IDE. If you are using VSCode just type: `code .`
