# 1.1.3

## cluster-registry-api
- SLT bugfix (#37)

## cluster-registry-client
- N/A

# 1.1.2

## cluster-registry-api
- Add SLT to check the update CRD operation (#34)

## cluster-registry-client
- N/A

# 1.0.2

## cluster-registry-api
- Fix rate limit middleware skipper (#32)

## cluster-registry-client
- N/A

# 1.0.1

## cluster-registry-api
- Add debug level configuration (#30)

## cluster-registry-client
- N/A

# 1.0.0

## cluster-registry-api
- Add rate limiter for /api/v1 (#27)
- Add request_id for each request (#28)

## cluster-registry-client
- N/A

# 0.2.1

## cluster-registry-api
- Registration timestamp (#25)
- Add the option to read the token from disk to the performance tests (#24)
- Fix annotation format (#22)
- Add support for tokens that does not have the 'spn' prefix (#23)
- Release tag fix for the performance tests image (#21)

## cluster-registry-client
- Fix annotation format (#22)

# 0.1.6

## cluster-registry-api
- Add /version endpoint
- Accept shortname
- Db test refactoring
- Improved /readyz endpoint
- Added /status check
- Added performance tests

## cluster-registry-client
- Use annotation mechanism to change controller behavior

# 0.1.5

## cluster-registry-api
- Add dynamodb global secondary index support

## cluster-registry-client
- N/A

# 0.1.3

## cluster-registry-api
- Fix metrics registration in Prometheus

## cluster-registry-client
- Fix metrics registration in Prometheus

# 0.1.2-2

## cluster-registry-api
- N/A

## cluster-registry-client
- Error handling for loading TLS files

# 0.1.2-1

## cluster-registry-api
- N/A

## cluster-registry-client
- Fix initializing CAData for in-cluster configuration

# 0.1.2

## cluster-registry-api
- N/A

## cluster-registry-client
- CertificateAuthorityData field auto-update on CRD create/update
