# Testing

To run unit tests for both api and client execute `make test`.

To run integration tests execute `make test-e2e`.

To run performance test locally you should run `make test-performance $API_URL=http://api-url/ NR_OF_SECS=10 API_TOKEN=$token`. The `API_URL` and `NR_OF_SECS` have the defaults localhost and 10s, but the `API_TOKEN` is mandatory.

To generate a local token you can use the following code:
```
	appConfig, _ := config.LoadApiConfig()
	jwtToken := jwt.GenerateSignedToken(appConfig, false, "test/testdata/dummyRsaPrivateKey.pem", "", jwt.Claim{})

	fmt.Println(jwtToken)
```