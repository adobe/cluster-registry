SHELL=/usr/bin/env bash -o pipefail

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
ifeq ($(GOARCH),arm)
	ARCH=armv7
else
	ARCH=$(GOARCH)
endif

GO_PKG=github.com/adobe/cluster-registry
TAG?=$(shell git rev-parse --short HEAD)
VERSION?=$(shell cat VERSION | tr -d " \t\n\r")

# The ldflags for the go build process to set the version related data.
GO_BUILD_LDFLAGS=\
	-s \
	-X main.Version=$(VERSION) \
	-X main.Revision=$(BUILD_REVISION)  \
	-X main.BuildUser=$(BUILD_USER) \
	-X main.BuildDate=$(BUILD_DATE) \
	-X main.Branch=$(BUILD_BRANCH)

GO_BUILD_RECIPE=\
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	CGO_ENABLED=0 \
	go build -ldflags="$(GO_BUILD_LDFLAGS)"

PKGS = $(shell go list ./pkg/...)
PKGS += $(shell go list ./cmd/...)

.PHONY: all
all: format generate build test test-e2e

.PHONY: clean
clean:
	# Remove all files and directories ignored by git.
	git clean -Xfd .


############
# Building #
############

.PHONY: build
build: build-apiserver build-client

.PHONY: build-apiserver
build-apiserver:
	$(GO_BUILD_RECIPE) -o cluster-registry-apiserver cmd/apiserver/apiserver.go

.PHONY: build-client
build-client:
	$(GO_BUILD_RECIPE) -o cluster-registry-client cmd/client/client.go

.PHONY: release
release:
	./hack/release.sh

.PHONY: image
image: GOOS := linux
image: .hack-apiserver-image .hack-client-image

.hack-apiserver-image: cmd/apiserver/Dockerfile build-apiserver
	docker build -t $(IMAGE_APISERVER):$(TAG) -f cmd/apiserver/Dockerfile .
	touch $@

.hack-client-image: cmd/client/Dockerfile build-client
	docker build -t $(IMAGE_CLIENT):$(TAG) -f cmd/client/Dockerfile .
	touch $@

.PHONY: update-go-deps
update-go-deps:
	for m in $$(go list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do \
		go get $$m; \
	done
	@echo "Don't forget to run 'make tidy'"

.PHONY: build-performance
build-performance:
	docker build -t ${IMAGE_PERFORMANCE_TESTS}:$(TAG) -f test/performance/Dockerfile .

.PHONY: release-performance
release-performance:
	./test/performance/scripts/release.sh

##############
# Formatting #
##############

.PHONY: format
format: go-fmt go-vet go-lint go-sec check-license

.PHONY: go-fmt
go-fmt:
	@echo 'Formatting go code...'
	@gofmt -s -w .
	@echo 'Not formatiing issues found in go codebase!'


.PHONY: check-license
check-license:
	./hack/check_license.sh


.PHONY: go-lint
go-lint: golangci-lint
	@echo 'Linting go code...'
	$(GOLANGCI_LINT) run -v --timeout 5m
	@echo 'Not linting issues found in go codebase!'

.PHONY: lint-fix
lint-fix: golangci-lint
	$(GOLANGCI_LINT) run --fix

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
golangci-lint:
	@[ -f $(GOLANGCI_LINT) ] || { \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) v1.41.1 ;\
	}

GOSEC = $(shell pwd)/bin/gosec
go-sec: ## Inspects source code for security problems
	@if ! command -v gosec >/dev/null 2>&1 ; then $(call go-get-tool,$(GOSEC),github.com/securego/gosec/v2/cmd/gosec); fi
	@echo 'Checking source code for security problems...'
	$(GOSEC)  ./pkg/...
	@echo 'No security problems found in go codebase!'

go-vet: ## Identifying subtle source code issues
	@echo 'Vetting go code...'
	@go vet $(shell pwd)/pkg/apiserver/...
	@echo 'Not issues found in go codebase!'


###########
# Testing #
###########

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin

.PHONY: test
test:
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -race $(TEST_RUN_ARGS) -short $(PKGS) -count=1 -cover -v

.PHONY: test-e2e
test-e2e:
	$(shell pwd)/local/setup.sh
	@. local/.env.local && go test -race github.com/adobe/cluster-registry/test/e2e -count=1 -v
	$(shell pwd)/local/cleanup.sh

## Make sure you have set the APISERVER_AUTH_TOKEN env variable
## Use PERFORMANCE_TEST_TIME env var to set the benchmark time per endpoint
## Use APISERVER_ENDPOINT env var to set the endpoint for the benchmark
.PHONY: test-performance
test-performance: ## Outputs requests/s, average req time, 99.9th percentile req time
	@. local/.env.local \
		&& export APISERVER_ENDPOINT=${APISERVER_ENDPOINT} \
		&& test/performance/scripts/run_bech_container.sh


###############
# Development #
###############

CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
MANAGER_ROLE ?= "cluster-registry"

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=$(MANAGER_ROLE) webhook paths="$(shell pwd)/pkg/api/..." output:crd:artifacts:config=$(shell pwd)/config/crd/bases output:rbac:artifacts:config=$(shell pwd)/config/rbac

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="$(shell pwd)/hack/boilerplate.go.txt" paths="$(shell pwd)/pkg/api/..."

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
	@[ -f $(1) ] || { \
	set -e; \
	TMP_DIR=$$(mktemp -d); \
	cd $$TMP_DIR; \
	go mod init tmp; \
	echo "Downloading $(2)"; \
	GOBIN=$(shell pwd)/bin go get $(2); \
	rm -rf $$TMP_DIR; \
	}
endef

# -------------
# DOCUMENTATION
# -------------
swagger:
	swag init --parseDependency --parseInternal --parseDepth 2 -g cmd/apiserver/apiserver.go --output pkg/apiserver/docs/
