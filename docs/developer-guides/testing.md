# Testing



To run unit tests for both api and client execute `make test`.

To run integration tests execute `make test-e2e`.

#### Performance tests

To run performance test locally you should run:

`. local/.env.local` To set the IMAGE_PERFORMANCE_TESTS its TAG and the container NETWORK.
`make test-performance $APISERVER_ENDPOINT=http://api-url/ PERFORMANCE_TEST_TIME=10 APISERVER_AUTH_TOKEN=$token`.

The `APISERVER_ENDPOINT` and `PERFORMANCE_TEST_TIME` have the defaults http://localhost:8080 and 10s, but the `APISERVER_AUTH_TOKEN` is mandatory.

To generate a local token you can use the following code:
```
	appConfig, _ := config.LoadApiConfig()
	jwtToken := jwt.GenerateSignedToken(appConfig, false, "test/testdata/dummyRsaPrivateKey.pem", "", jwt.Claim{})

	fmt.Println(jwtToken)
```