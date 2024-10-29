# Local Development Setup

To contribute to Cluster Registry, you need to set up a local environment.

## Initial setup

Cluster Registry development stack consist of the following applications that runs in Docker containers:

* cluster-registry-api
* cluster-registry-client
* cluster-registry-sync-manager
* local aws sqs
* local aws dynamoDb
* local k8s cluster created with kind
* local oidc mocking provider

If you want to test one of the Cluster Registry components call directly setup script by specifying to/not to run that component:

* run the stack without cluster-registry-api: `API=false make setup`
* run the stack without cluster-registry-client: `CLIENT=false make setup`
* run the stack without cluster-registry-sync-manager : `SYNC_MANAGER=false make setup`
* run the stack with all components: `make setup`

To clean up your local setup run:
`make clean`

Open your IDE. If you are using VSCode just type: `code .`
