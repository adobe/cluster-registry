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
	-X $(GO_PKG)/version.Revision=$(BUILD_REVISION)  \
	-X $(GO_PKG)/version.BuildUser=$(BUILD_USER) \
	-X $(GO_PKG)/version.BuildDate=$(BUILD_DATE) \
	-X $(GO_PKG)/version.Branch=$(BUILD_BRANCH) \
	-X $(GO_PKG)/version.Version=$(VERSION)

GO_BUILD_RECIPE=\
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	CGO_ENABLED=0 \
	go build -ldflags="$(GO_BUILD_LDFLAGS)"

API_PKGS = $(shell go list ./pkg/api/...)
API_PKGS += $(shell go list ./cmd/api/...)
CC_PKGS = $(shell go list ./pkg/cc/...)
CC_PKGS += $(shell go list ./cmd/cc/...)

.PHONY: all
all: format generate build test

.PHONY: clean
clean:
	# Remove all files and directories ignored by git.
	git clean -Xfd .


############
# Building #
############

.PHONY: build
build: # TODO


##############
# Formatting #
##############

.PHONY: format
format: go-fmt jsonnet-fmt check-license shellcheck

.PHONY: go-fmt
go-fmt:
	gofmt -s -w .

.PHONY: check-license
check-license:
	./hack/check_license.sh

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint
	$(GOLANGCI_LINT) run --fix

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
golangci-lint:
	@[ -f $(GOLANGCI_LINT) ] || { \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) v1.41.1 ;\
	}


###########
# Testing #
###########

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin

.PHONY: test
test: test-api test-cc

.PHONY: test-api
test-api: source-env
	source $(shell pwd)/.env.sample; go test -race $(TEST_RUN_ARGS) -short $(API_PKGS) -count=1 -v

.PHONY: test-cc
test-cc: source-env
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -race $(TEST_RUN_ARGS) -short $(CC_PKGS) -count=1 -v

source-env:
	source $(shell pwd)/.env.sample


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
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=$(MANAGER_ROLE) webhook paths="$(shell pwd)/pkg/cc/..." output:crd:artifacts:config=$(shell pwd)/config/crd/bases output:rbac:artifacts:config=$(shell pwd)/config/rbac

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="$(shell pwd)/hack/boilerplate.go.txt" paths="$(shell pwd)/pkg/cc/..."

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(shell pwd)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef