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

### Synthetic tests

There are two synthetic tests for Cluster Registry ran in a goroutine by a service and serves metrics using the `/metrics` endpoint for Prometheus to scrape. One e2e test that checks if an update of the `cluster` custom resource gets propagated to the CR API, and another test that checks the two main endpoints: `/api/v1/clusters/[cluster]` and `/api/v1/clusters`.
#### E2e synthetic test

```mermaid
sequenceDiagram
	participant TestService
	participant ClusterRegistryClient
	participant ClusterRegistryAPI
	participant CRD
	participant Prometheus
	Note left of TestService: CR API Token
	TestService->>CRD: Modify `update-slt` tag
	ClusterRegistryClient->>CRD: Check crd change
	ClusterRegistryClient->>ClusterRegistryAPI: Push change to database
	TestService->>ClusterRegistryAPI: Check change in database
	Prometheus->>TestService: Scrape /metrics endpoint
```

The test consists of the following stages with it's following logs:
1. Get a token to authenticate to Cluster Registry.
2. Make an update on the cluster custom object by adding a new `Tag` named `update-slt` with the value `Tick` or `Tack`. Logs in order:
    - `updating the Cluster Registry CRD...`
    - `changing 'update-slt' tag value`
    - `cluster Registry CRD updated!`
3. Query the Cluster Registry API (using the endpoint `/api/v1/clusters/[cluster-name]`) and check for the update. Logs in order:
    - `waiting for the Cluster Registry API to update the database...`
    - `checking the API for the update (check 1/3)... (reties 3 times at a 11s interval)`
    - `update confirmed`

#### Get endpoints synthetic test

```mermaid
sequenceDiagram
	participant TestService
	participant ClusterRegistryAPI
	participant Prometheus
	Note left of TestService: CR API Token
	TestService->>ClusterRegistryAPI: GET request on one of the endpoints
	Prometheus->>TestService: Scrape /metrics endpoint
```

The test consists of the following stages with it's following logs:
1. Get a token to authenticate to Cluster Registry.
2. Make a GET on one of the endpoints. Logs example:
   - `timing the request that gets a cluster...`
   - `timing completed for the request that gets a cluster: took 0.017150s`
3. Check response code to be `200` and validate the payload