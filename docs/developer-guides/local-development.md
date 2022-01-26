# Local Development Setup

To contribute to Cluster Registry, you need to set up a local environment.

## Initial setup

Cluster Registry development stack consist of the following applications that runs in Docker containers:

* cluster-registry-api
* cluster-registry-client
* aws sqs
* aws dynamoDb
* a local k8s cluster created with kind

If you want to test one of the Cluster Registry components call directly setup script by specifying to/not to run that component:

* run the stack without cluster registry api: `local/setup.sh 0 1`
* run the stack without cluster registry client: `local/setup.sh 0 1`

Open your IDE. If you are using VSCode just type: `code .`
