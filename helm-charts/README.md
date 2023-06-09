# Cluster Registry Client Chart

Helm chart for Cluster Registry Client.

## Prerequisites for installing the chart

**Notes**
> Secret management must be handled by the user. Secrets must be created (either by kubectl or by a secret management tool) beforehand in order for the chart deployment to be successful.
> This step is mandatory for the application to run succesfully.

Required secret name is `cluster-registry-aws` and required keys are: `AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, SQS_AWS_REGION, SQS_ENDPOINT, SQS_QUEUE_NAME`

Example of a secret name and corresponding required keys (can also be found in [deployment.yaml](cluster-registry-client/templates/deployment.yaml) file):
```yaml
apiVersion: v1
data:
  AWS_ACCESS_KEY_ID: <AWS_ACCESS_KEY_ID_VALUE>
  AWS_SECRET_ACCESS_KEY: <AWS_SECRET_ACCESS_KEY_VALUE>
  SQS_AWS_REGION: <SQS_AWS_REGION_VALUE>
  SQS_ENDPOINT: <SQS_ENDPOINT_VALUE>
  SQS_QUEUE_NAME: <SQS_QUEUE_NAME_VALUE>
kind: Secret
metadata:
  name: cluster-registry-aws
  namespace: <namespace>
type: Opaque
```

## Chart configuration
To see a full list if configurable parameters, check the table generated in the [README.md](cluster-registry-client/README.md) file.

### Deploying the chart locally
```bash
 helm install cluster-registry-client . --values values.yaml --namespace <namespace>
```
