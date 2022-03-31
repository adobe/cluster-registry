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