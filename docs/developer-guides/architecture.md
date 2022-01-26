# Architecture

Cluster Registry has a push architecture, with a CRD being created in each cluster.
The CRD is created at the cluster build.

Once the CRD is created in a cluster, it is automatically shipped to the central API via Cluster Registry Client.

High Level Design
The Cluster Register has couple of components:

* this CRD will be populated from cluster build/upgrade CD tool - reads data from cluster config and updates cluster CRD attributes.
* cluster-registry-client (in-cluster light k8s operator) that watches for CRD changes and send messages to a AWS SQS queue. It also expose a webhook in order to receive information from Observability (ex. cluster capacity).
* cluster-registry-api reads the information related to all Ethos clusters from the AWS SQS and will be centralize it into a DynamoDB database and presented via REST API.
  * The API exposes only GET HTTP methods
  * All API requests are authenticated. As auth Provider cluster-registry-api can use any OIDC compliant provider (Okta, AzureAD).

The following diagram describes this process.

![architecture](https://lucid.app/publicSegments/view/4b7b1961-92a4-484d-b9af-534fa1be3ba7/image.png)
