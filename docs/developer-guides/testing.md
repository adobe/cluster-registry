# Testing



To run unit tests for both api and client execute `make test`.

To run integration tests execute `make test-e2e`.

#### Performance tests

To run performance test locally you should run:

`make test-performance APISERVER_AUTH_TOKEN=$token APISERVER_ENDPOINT=http://api-url/ PERFORMANCE_TEST_TIME=10`.

It will take the defaults from the `local/.env.local` file but it will overwrite them form the local env if the environment variables exists.
If the file and the env vars are missing the `APISERVER_ENDPOINT` and `PERFORMANCE_TEST_TIME` have the defaults `http://localhost:8080` and `10` seconds, but the `APISERVER_AUTH_TOKEN` is mandatory.

To generate a local token you can use the following code:
```
	import (
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/test/jwt"

	"fmt"
)

func main() {

	appConfig, _ := config.LoadApiConfig()
	jwtToken := jwt.GenerateSignedToken(appConfig, false, "test/testdata/dummyRsaPrivateKey.pem", "", jwt.Claim{})

	fmt.Println(jwtToken)
}
```

#### SLT test

This is an end to end test that consists of a service that serves the /metrics endpoint for Prometheus to scrape the test metrics, and the test itself that runs in a loop and updates the metrics.

```mermaid
sequenceDiagram
    participant SLTService
    participant ClusterRegistryClient
	participant ClusterRegistryAPI
	participant CRD
	participant Prometheus
	Note left of SLTService: CR API Token
    SLTService->>CRD: Modify `update-slt` tag
    ClusterRegistryClient->>CRD: Check crd change
    ClusterRegistryClient->>ClusterRegistryAPI: Push change to database
    SLTService->>ClusterRegistryAPI: Check change in database
	Prometheus->>SLTService: Scrape /metrics endpoint
```

The test and consists of the following stages with it's following logs:
1. Get a token to authenticate to Cluster Registry.
2. Makes an update on the cluster custom object by adding new `Tag` named `update-slt` with the value `Tick` or `Tack`. Logs in order:
    - `Updating the Cluster Registry CRD...`
    - `Changing 'update-slt' tag value`
    - `Cluster Registry CRD updated!`
3. Query the Cluster Registry API to check for the update at the endpoint `/api/v1/clusters/[cluster-name]`. Logs in order:
    - `Waiting for the Cluster Registry API to update the database...`
    - `Checking the API for the update (check 1/3)... (reties 3 times at a 11s interval)`
    - `Update confirmed`
