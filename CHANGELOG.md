# 1.6.5

### 🐛 Bug Fixes

- Fix ChargedBack field marshaling issue (#257)
- *(deps)* Update kubernetes packages to v0.31.1 (#248)
- *(deps)* Update module github.com/prometheus/client_golang to v1.20.4 (#251)
- *(deps)* Update k8s.io/utils digest to 49e7df5 (#252)


# 1.6.4

### 🐛 Bug Fixes

- Add TLS config to redis client (#254)


# 1.6.3

### 🚀 Features

- Add Metrics to the sync manager (#244)
- Add BQU fields (#241)


### 🐛 Bug Fixes

- Temporarily removing Codecov workflow (#247)
- fix(deps): update module golang.org/x/net to v0.29.0 (#242)
- fix(deps): update module github.com/prometheus/client_golang to v1.20.3 (#229)
- fix(deps): update module github.com/testcontainers/testcontainers-go to v0.33.0 (#230)
- fix(deps): update k8s.io/utils digest to 702e33f (#231)
- fix(deps): update golang.org/x/exp digest to e7e105d (#233)
- dependabot(deps): bump github.com/onsi/gomega from 1.34.1 to 1.34.2 (#239)
- fix(deps): update module dario.cat/mergo to v1.0.1 (#227)


# 1.6.2

### 🐛 Bug Fixes

- Fix issue with sync-manager reconciliation logic (#223)
- *(deps)* Update module github.com/prometheus/client_golang to v1.20.0 (#221)
- *(deps)* Bump controller-gen version to 0.16.1 (#224)


# 1.6.1

### 🐛 Bug Fixes

- cluster-registry-sync-manager nil pointer dereference (#219)


# 1.6.0

### 🚀 Features

- cluster-registry-sync-manager (#177)
- cluster-registry-api caching (#202)
- Add cluster-registry-sync-manager chart (#211)

### 🐛 Bug Fixes

- *(deps)* Update module github.com/aws/aws-sdk-go to v1.55.3 (#165)
- *(deps)* Update golang.org/x/exp digest to 8a7402a (#190)
- *(deps)* Update kubernetes packages to v0.30.3 (#154)
- *(deps)* Update module github.com/azure/go-autorest/autorest/azure/auth to v0.5.13 (#166)
- *(deps)* Update module github.com/testcontainers/testcontainers-go to v0.32.0 (#182)
- *(deps)* Update module github.com/go-jose/go-jose/v3 to v4 (#163)
- *(deps)* Update module github.com/onsi/ginkgo to v2 (#164)
- *(deps)* Update k8s.io/utils digest to 18e509b (#191)
- *(deps)* Update module github.com/go-jose/go-jose/v3 to v4 (#193)
- *(deps)* Update module github.com/coreos/go-oidc/v3 to v3.11.0 (#192)
- *(deps)* Update module github.com/onsi/ginkgo to v2 (#195)
- *(deps)* Update module github.com/go-jose/go-jose/v3 to v4 (#194)
- *(deps)* Update module github.com/onsi/gomega to v1.34.1 (#199)
- *(deps)* Update module github.com/aws/aws-sdk-go to v1.55.4 (#198)
- *(deps)* Update module github.com/go-jose/go-jose/v3 to v4 (#196)
- *(deps)* Update module github.com/redis/go-redis/v9 to v9.6.1 (#206)
- *(deps)* Update module golang.org/x/net to v0.28.0 (#205)
- *(deps)* Update module github.com/onsi/ginkgo to v2 (#197)
- *(deps)* Update module github.com/go-jose/go-jose/v3 to v4 (#200)
- *(deps)* Update module github.com/go-jose/go-jose/v3 to v4 (#208)
- *(deps)* Update module github.com/onsi/ginkgo to v2 (#207)
- *(deps)* Update golang.org/x/exp digest to 0cdaa3a (#212)
- *(deps)* Bump github.com/aws/aws-sdk-go from 1.55.4 to 1.55.5 (#203)


# 1.5.8

## cluster-registry-api

- Chargeback - Swagger docs (#175)

## cluster-registry-client

- Chargeback - Swagger docs (#175)

# 1.5.7

## cluster-registry-api

- Make chargeback fields as optional (#167)
- fix(deps): update module github.com/go-jose/go-jose/v3 to v4 (#162)
- fix(deps): update module github.com/onsi/ginkgo to v2 (#161)
- fix(deps): update module github.com/go-jose/go-jose/v3 to v4 (#157)
- fix(deps): update module github.com/onsi/ginkgo to v2 (#158)
- fix(deps): update k8s.io/utils digest to fe8a2dd (#151)
- fix(deps): update module github.com/go-logr/logr to v1.4.2 (#152)
- Add renovate.json (#134)
- dependabot(deps): bump github.com/labstack/echo/v4 from 4.11.4 to 4.12.0 (#141)
- dependabot(deps): bump github.com/onsi/gomega from 1.32.0 to 1.33.1 (#145)
- dependabot(deps): bump github.com/prometheus/client_golang (#146)
- dependabot(deps): bump github.com/testcontainers/testcontainers-go (#147)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.16 to 1.53.10 (#150)

## cluster-registry-client

- Make chargeback fields as optional (#167)
- fix(deps): update module github.com/go-jose/go-jose/v3 to v4 (#162)
- fix(deps): update module github.com/onsi/ginkgo to v2 (#161)
- fix(deps): update module github.com/go-jose/go-jose/v3 to v4 (#157)
- fix(deps): update module github.com/onsi/ginkgo to v2 (#158)
- fix(deps): update k8s.io/utils digest to fe8a2dd (#151)
- fix(deps): update module github.com/go-logr/logr to v1.4.2 (#152)
- Add renovate.json (#134)
- dependabot(deps): bump github.com/labstack/echo/v4 from 4.11.4 to 4.12.0 (#141)
- dependabot(deps): bump github.com/onsi/gomega from 1.32.0 to 1.33.1 (#145)
- dependabot(deps): bump github.com/prometheus/client_golang (#146)
- dependabot(deps): bump github.com/testcontainers/testcontainers-go (#147)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.16 to 1.53.10 (#150)

# 1.5.6

## cluster-registry-api

- dependabot(deps): bump golang.org/x/net from 0.22.0 to 0.24.0 (#131)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.11 to 1.51.16 (#132)

## cluster-registry-client

- Fix extra metrics handler (#135)
- dependabot(deps): bump golang.org/x/net from 0.22.0 to 0.24.0 (#131)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.11 to 1.51.16 (#132)

# 1.5.5

## cluster-registry-api

- Add NamespaceProfileInfraType field (#130)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.6 to 1.51.11 (#129)
- dependabot(deps): bump github.com/coreos/go-oidc/v3 from 3.9.0 to 3.10.0 (#128)
- dependabot(deps): bump github.com/onsi/gomega from 1.31.1 to 1.32.0 (#127)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.1 to 1.51.6 (#126)
- dependabot(deps): bump golang.org/x/net from 0.21.0 to 0.22.0 (#124)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.50.30 to 1.51.1 (#125)
- dependabot(deps): bump github.com/go-jose/go-jose/v3 from 3.0.2 to 3.0.3 (#123)
- dependabot(deps): bump github.com/testcontainers/testcontainers-go (#121)
- dependabot(deps): bump github.com/stretchr/testify from 1.8.4 to 1.9.0 (#120)
- dependabot(deps): bump github.com/prometheus/client_golang (#119)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.50.25 to 1.50.30 (#118)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.50.18 to 1.50.25 (#117)
- dependabot(deps): bump github.com/go-jose/go-jose/v3 from 3.0.1 to 3.0.2 (#116)
- Update dependencies (#110)
- dependabot(deps): bump github.com/testcontainers/testcontainers-go (#114)
- dependabot(deps): bump github.com/swaggo/swag from 1.16.1 to 1.16.3 (#113)
- dependabot(deps): bump sigs.k8s.io/yaml from 1.3.0 to 1.4.0 (#112)
- dependabot(deps): bump github.com/coreos/go-oidc/v3 from 3.6.0 to 3.9.0 (#111)

## cluster-registry-client

- Add NamespaceProfileInfraType field (#130)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.6 to 1.51.11 (#129)
- dependabot(deps): bump github.com/coreos/go-oidc/v3 from 3.9.0 to 3.10.0 (#128)
- dependabot(deps): bump github.com/onsi/gomega from 1.31.1 to 1.32.0 (#127)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.51.1 to 1.51.6 (#126)
- dependabot(deps): bump golang.org/x/net from 0.21.0 to 0.22.0 (#124)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.50.30 to 1.51.1 (#125)
- dependabot(deps): bump github.com/go-jose/go-jose/v3 from 3.0.2 to 3.0.3 (#123)
- dependabot(deps): bump github.com/testcontainers/testcontainers-go (#121)
- dependabot(deps): bump github.com/stretchr/testify from 1.8.4 to 1.9.0 (#120)
- dependabot(deps): bump github.com/prometheus/client_golang (#119)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.50.25 to 1.50.30 (#118)
- dependabot(deps): bump github.com/aws/aws-sdk-go from 1.50.18 to 1.50.25 (#117)
- dependabot(deps): bump github.com/go-jose/go-jose/v3 from 3.0.1 to 3.0.2 (#116)
- Update dependencies (#110)
- dependabot(deps): bump github.com/testcontainers/testcontainers-go (#114)
- dependabot(deps): bump github.com/swaggo/swag from 1.16.1 to 1.16.3 (#113)
- dependabot(deps): bump sigs.k8s.io/yaml from 1.3.0 to 1.4.0 (#112)
- dependabot(deps): bump github.com/coreos/go-oidc/v3 from 3.6.0 to 3.9.0 (#111)

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
