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

###############
# Local setup #
###############

SETUP_CMD = "./local/setup.sh"
ifeq ($(API),true)
	ifeq ($(CLIENT),)
		SETUP_CMD += "1 0"
	endif
else ifeq ($(API),)
	ifeq ($(CLIENT),true)
		SETUP_CMD += "0 1"
	endif
endif

.PHONY: clean
clean:
	@echo "Cleaning local environment..."
	./local/cleanup.sh

.PHONY: setup
setup:
	@echo "Running local setup..."
	@ $(SETUP_CMD)

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

.PHONY: build-slt
build-slt:
	docker build -t ${IMAGE_SLT}:$(TAG) -f test/slt/Dockerfile .

.PHONY: release-slt
release-slt:
	./test/slt/release.sh

##############
# Formatting #
##############

.PHONY: format-prereq
format-prereq:
	@[ -f $(GOSEC) ] || GOBIN=$(shell pwd)/bin go get "github.com/securego/gosec/v2/cmd/gosec";

.PHONY: format
format: format-prereq go-fmt go-vet go-lint go-sec check-license

.PHONY: go-fmt
go-fmt:
	@echo 'Formatting go code...'
	@gofmt -s -w .
	@echo 'No formating issues found in go codebase!'

.PHONY: check-license
check-license:
	./hack/check_license.sh

.PHONY: go-lint
go-lint: golangci-lint
	@echo 'Linting go code...'
	$(GOLANGCI_LINT) run -v --timeout 5m
	@echo 'No linting issues found in go codebase!'

.PHONY: lint-fix
lint-fix: golangci-lint
	$(GOLANGCI_LINT) run --fix

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GOLANGCI_LINT_VERSION = "v1.54.2"
golangci-lint:
	@[ -f $(GOLANGCI_LINT) ] || GOBIN=$(shell pwd)/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION);

GOSEC = $(shell pwd)/bin/gosec
.PHONY: go-sec
go-sec:
	@[ -f $(GOSEC) ] || GOBIN=$(shell pwd)/bin go install "github.com/securego/gosec/v2/cmd/gosec";
	@echo 'Checking source code for security problems...'
	$(GOSEC)  ./pkg/...
	@echo 'No security problems found in go codebase!'	

.PHONY: go-vet
go-vet:
	@echo 'Vetting go code and identify subtle source code issues...'
	@go vet $(shell pwd)/pkg/apiserver/...
	@echo 'No issues found in go codebase!'


###########
# Testing #
###########

KUBEBUILDER_ASSETS=$(shell pwd)/kubebuilder
K8S_VERSION=1.25.0

.PHONY: test
test:
	@[ -d $(KUBEBUILDER_ASSETS) ] || {\
		mkdir -p $(KUBEBUILDER_ASSETS);\
		curl -sSLo envtest-bins.tar.gz "https://go.kubebuilder.io/test-tools/$(K8S_VERSION)/$(GOOS)/$(GOARCH)";\
		tar -C $(KUBEBUILDER_ASSETS) --strip-components=1 -zvxf envtest-bins.tar.gz;\
	}
	KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS)/bin go test -race $(TEST_RUN_ARGS) -short $(PKGS) -count=1 -cover -v

.PHONY: test-e2e
test-e2e:
	$(shell pwd)/local/setup.sh
	@. local/.env.local && go test -race github.com/adobe/cluster-registry/test/e2e -count=1 -v
	# $(shell pwd)/local/cleanup.sh

## Make sure you have set the APISERVER_AUTH_TOKEN env variable.
## Use PERFORMANCE_TEST_TIME env var to set the benchmark time per endpoint.
## Use APISERVER_ENDPOINT env var to set the endpoint for the benchmark.
## The env variables from the local env will overwrite th local/.env.local ones.
.PHONY: test-performance
test-performance: ## Outputs requests/s, average req time, 99.9th percentile req time
	@. local/.env.local \
		&& export LOCAL_ENV_APISERVER_ENDPOINT=${APISERVER_ENDPOINT} \
		&& export LOCAL_ENV_APISERVER_AUTH_TOKEN=${APISERVER_AUTH_TOKEN} \
		&& export LOCAL_ENV_PERFORMANCE_TEST_TIME=${PERFORMANCE_TEST_TIME} \
		&& export LOCAL_ENV_IMAGE_PERFORMANCE_TESTS=${IMAGE_PERFORMANCE_TESTS} \
		&& export LOCAL_ENV_TAG=${TAG} \
		&& export LOCAL_ENV_NETWORK=${NETWORK} \
		&& test/performance/scripts/run_bech_container.sh


###############
# Development #
###############

CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
MANAGER_ROLE ?= "cluster-registry"

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	@[ -f $(CONTROLLER_GEN) ] || GOBIN=$(shell pwd)/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.13.0

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	@[ -f $(KUSTOMIZE) ] || GOBIN=$(shell pwd)/bin go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.5

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=$(MANAGER_ROLE) webhook paths="$(shell pwd)/pkg/api/..." output:crd:artifacts:config=$(shell pwd)/config/crd/bases output:rbac:artifacts:config=$(shell pwd)/config/rbac

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="$(shell pwd)/hack/boilerplate.go.txt" paths="$(shell pwd)/pkg/api/..."


#################
# Documentation #
#################

SWAGGER_CLI = $(shell pwd)/bin/swag
swagger:
	@[ -f $(SWAGGER_CLI) ] || GOBIN=$(shell pwd)/bin go install github.com/swaggo/swag/cmd/swag@v1.8.12
	$(SWAGGER_CLI) init --parseDependency --parseInternal --parseDepth 2 -g cmd/apiserver/apiserver.go --output pkg/apiserver/docs/
