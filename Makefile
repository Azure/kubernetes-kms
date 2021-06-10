ORG_PATH=github.com/Azure
PROJECT_NAME := kubernetes-kms
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"

REGISTRY_NAME ?= upstreamk8sci
REPO_PREFIX ?= oss/azure/kms
REGISTRY ?= $(REGISTRY_NAME).azurecr.io/$(REPO_PREFIX)
LOCAL_REGISTRY_NAME ?= kind-registry
LOCAL_REGISTRY_PORT ?= 5000
IMAGE_NAME ?= keyvault
IMAGE_VERSION ?= v0.0.12
IMAGE_TAG := $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
CGO_ENABLED_FLAG := 0

# build variables
BUILD_VERSION_VAR := $(REPO_PATH)/pkg/version.BuildVersion
BUILD_DATE_VAR := $(REPO_PATH)/pkg/version.BuildDate
BUILD_DATE := $$(date +%Y-%m-%d-%H:%M)
GIT_VAR := $(REPO_PATH)/pkg/version.GitCommit
GIT_HASH := $$(git rev-parse --short HEAD)

GO_FILES=$(shell go list ./... | grep -v /test/e2e)
TOOLS_MOD_DIR := ./tools
TOOLS_DIR := $(abspath ./.tools)

# docker env var
DOCKER_BUILDKIT = 1
export DOCKER_BUILDKIT

# Testing var
KIND_VERSION ?= 0.11.0
KUBERNETES_VERSION ?= v1.21.1
BATS_VERSION ?= 1.2.1

GO_BUILD_OPTIONS := --tags "netgo osusergo"  -ldflags "-s -X $(BUILD_VERSION_VAR)=$(IMAGE_VERSION) -X $(GIT_VAR)=$(GIT_HASH) -X $(BUILD_DATE_VAR)=$(BUILD_DATE) -extldflags '-static'"

$(TOOLS_DIR)/golangci-lint: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint
	$(TOOLS_DIR)/golangci-lint run --timeout=5m -v

.PHONY: build
build:
	$Q GOOS=linux CGO_ENABLED=0 go build $(GO_BUILD_OPTIONS) -o _output/kubernetes-kms ./cmd/server/

.PHONY: build-darwin
build-darwin:
	$Q GOOS=darwin CGO_ENABLED=0 go build $(GO_BUILD_OPTIONS) -o _output/kubernetes-kms ./cmd/server/

build-image: clean build
	$Q docker build -t $(IMAGE_TAG) .

push-image: build-image
	$Q docker push $(IMAGE_TAG)

.PHONY: clean unit-test integration-test

clean:
	$Q rm -rf _output/

authors:
	$Q git log --all --format='%aN <%cE>' | sort -u  | sed -n '/github/!p' > GITAUTHORS
	$Q cat AUTHORS GITAUTHORS  | sort -u > NEWAUTHORS
	$Q mv NEWAUTHORS AUTHORS
	$Q rm -f NEWAUTHORS
	$Q rm -f GITAUTHORS

integration-test:
	$Q sudo GOPATH=$(GOPATH) go test -v -count=1 -failfast github.com/Azure/kubernetes-kms/tests/client

unit-test:
	go test -race -v -count=1 -failfast `go list ./... | grep -v client`

.PHONY: mod
mod:
	@go mod tidy

## --------------------------------------
## E2E Testing
## --------------------------------------
e2e-install-prerequisites:
	# Download and install kind
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 --output kind && chmod +x kind && sudo mv kind /usr/local/bin/
	# Download and install kubectl
	curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl && chmod +x ./kubectl && sudo mv kubectl /usr/local/bin/
	# Download and install bats
	curl -sSLO https://github.com/bats-core/bats-core/archive/v${BATS_VERSION}.tar.gz && tar -zxvf v${BATS_VERSION}.tar.gz && sudo bash bats-core-${BATS_VERSION}/install.sh /usr/local

.PHONY: install-soak-prerequisites
install-soak-prerequisites: e2e-install-prerequisites
	# Download and install node-shell
	curl -LO https://github.com/kvaps/kubectl-node-shell/raw/master/kubectl-node_shell && chmod +x ./kubectl-node_shell && sudo mv ./kubectl-node_shell /usr/local/bin/kubectl-node_shell

e2e-setup-kind: setup-local-registry
	./scripts/setup-kind-cluster.sh &
	./scripts/connect-registry.sh &
	sleep 90s

.PHONY: setup-local-registry
setup-local-registry:
	./scripts/setup-local-registry.sh

e2e-generate-manifests:
	@mkdir -p tests/e2e/generated_manifests
	envsubst < tests/e2e/azure.json > tests/e2e/generated_manifests/azure.json
	envsubst < tests/e2e/kms.yaml > tests/e2e/generated_manifests/kms.yaml

e2e-delete-kind:
	# delete kind e2e cluster created for tests
	kind delete cluster --name kms

e2e-test:
	# Run test suite with kind cluster
	bats -t tests/e2e/test.bats
