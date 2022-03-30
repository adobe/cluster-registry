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
