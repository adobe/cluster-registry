# 1.5.5

## cluster-registry-api

- Add NamespaceProfileInfraType field (#108)

## cluster-registry-client

- Add NamespaceProfileInfraType field (#108)

# 1.5.4

## cluster-registry-api

- Add maintenanceGroup and argoInstance fields (#107)
- chargedBack field added (#95)

## cluster-registry-client

- Add maintenanceGroup and argoInstance fields (#107)
- chargedBack field added (#95)
- Fix webhook tests (#96)

# 1.5.3

## cluster-registry-api

- Add AvailabilityZones field (#92)

## cluster-registry-client

- Add AvailabilityZones field (#92)

# 1.5.2

## cluster-registry-client

- Add a check for available GVKs using a discovery client. If a configured GVK isn't installed on the cluster it will skip it instead of returning an error.

- Improve field parsing for `servicemetadatawatcher.spec.watchedServiceObjects.watchedFields.src`

# 1.5.1

## cluster-registry-api

- Add OidcIssuer extra field (#87) 

## cluster-registry-client

- Add OidcIssuer extra field (#87)

# 1.5.0

## cluster-registry-api

- New endpoints for service metadata feature

## cluster-registry-client

- Service Metadata feature support

# 1.4.2

## cluster-registry-api

- Add recommended labels to Helm chart (#71)
- Add chargebackBusinessUnit field (#74)
- Remove k8sInfraRelease field (#74)
- Bump Go version to 1.21 (#74)

## cluster-registry-client

- Add recommended labels to Helm chart (#71)
- Add chargebackBusinessUnit field (#74)
- Remove k8sInfraRelease field (#74)
- Bump Go version to 1.21 (#74)
- Refactor deprecated clientConfig (#74)

# 1.4.1

## cluster-registry-api

- N/A

## cluster-registry-client

- Convert to helm (#57)

# 1.4.0

## cluster-registry-api

- Add patch endpoint (#58)

## cluster-registry-client

- N/A

# 1.3.1

## cluster-registry-api
- N/A

## cluster-registry-client
- Update local crds (#50)
- Fix generate CRD (#55)

# 1.3.0

## cluster-registry-api
- Add Capacity fields for Capacity Forecaster (#49)
- Add a make target for local setup (#48)
- Improved cluster filtering (#47)
- Add documentation to the e2e test (#41)

## cluster-registry-client
- N/A

# 1.2.1

## cluster-registry-api
- Update the metrics buckets for request duration tests (#45)

## cluster-registry-client
- N/A

# 1.2.0

## cluster-registry-api
- Added new request time test for the SLTs (#42)

## cluster-registry-client
- Fixed webhook tests (#43)
- Updated the webhook with reties (#43)

# 1.1.4

## cluster-registry-api
- Fix swagger #39

## cluster-registry-client
- N/A

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
