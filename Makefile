ORG_PATH=github.com/Azure
PROJECT_NAME := kubernetes-kms
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"

REGISTRY_NAME ?= upstreamk8sci
REPO_PREFIX ?= oss/azure/kms
REGISTRY ?= $(REGISTRY_NAME).azurecr.io/$(REPO_PREFIX)
LOCAL_REGISTRY_NAME ?= kind-registry
LOCAL_REGISTRY_PORT ?= 5000
IMAGE_NAME ?= keyvault
IMAGE_VERSION ?= v0.7.0
IMAGE_TAG := $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
CGO_ENABLED_FLAG := 0

# build variables
BUILD_VERSION_VAR := $(REPO_PATH)/pkg/version.BuildVersion
BUILD_DATE_VAR := $(REPO_PATH)/pkg/version.BuildDate
BUILD_DATE := $$(date +%Y-%m-%d-%H:%M)
GIT_VAR := $(REPO_PATH)/pkg/version.GitCommit
GIT_HASH := $$(git rev-parse --short HEAD)
LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(IMAGE_VERSION) -X $(GIT_VAR)=$(GIT_HASH)"

GO_FILES=$(shell go list ./... | grep -v /test/e2e)
TOOLS_MOD_DIR := ./tools
TOOLS_DIR := $(abspath ./.tools)

# docker env var
DOCKER_BUILDKIT = 1
export DOCKER_BUILDKIT

# Testing var
KIND_VERSION ?= 0.27.0
KUBERNETES_VERSION ?= v1.32.3
BATS_VERSION ?= 1.4.1

## --------------------------------------
## Linting
## --------------------------------------

$(TOOLS_DIR)/golangci-lint: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint
	$(TOOLS_DIR)/golangci-lint run --timeout=5m -v

## --------------------------------------
## Images
## --------------------------------------

ALL_LINUX_ARCH ?= amd64 arm64
# Output type of docker buildx build
OUTPUT_TYPE ?= type=registry

BUILDX_BUILDER_NAME ?= img-builder
QEMU_VERSION ?= 5.2.0-2
# The architecture of the image
ARCH ?= amd64

.PHONY: build
build:
	go build -a -ldflags $(LDFLAGS) -o _output/kubernetes-kms ./cmd/server/

.PHONY: docker-init-buildx
docker-init-buildx:
	@if ! docker buildx ls | grep $(BUILDX_BUILDER_NAME); then \
		docker run --rm --privileged mirror.gcr.io/multiarch/qemu-user-static:$(QEMU_VERSION) --reset -p yes; \
		docker buildx create --name $(BUILDX_BUILDER_NAME) --use; \
		docker buildx inspect $(BUILDX_BUILDER_NAME) --bootstrap; \
	fi

.PHONY: docker-build
docker-build:
	docker buildx build \
		--build-arg LDFLAGS=$(LDFLAGS) \
		--no-cache \
		--platform="linux/$(ARCH)" \
		--output=$(OUTPUT_TYPE) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)-linux-$(ARCH)  . \
		--progress=plain; \

	@if [ "$(ARCH)" = "amd64" ] && [ "$(OUTPUT_TYPE)" = "type=docker" ]; then \
		docker tag $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)-linux-$(ARCH) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION); \
	fi

.PHONY: docker-build-all
docker-build-all:
	@for arch in $(ALL_LINUX_ARCH); do \
		$(MAKE) ARCH=$${arch} docker-build; \
	done

.PHONY: docker-push-manifest
docker-push-manifest:
	docker manifest create --amend $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION) $(foreach arch,$(ALL_LINUX_ARCH),$(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)-linux-$(arch)); \
	for arch in $(ALL_LINUX_ARCH); do \
		docker manifest annotate --os linux --arch $${arch} $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)-linux-$${arch}; \
	done; \
	docker manifest push --purge $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION); \

## --------------------------------------
## Testing
## --------------------------------------

.PHONY: integration-test
integration-test:
	go test -v -count=1 -failfast github.com/Azure/kubernetes-kms/tests/client

.PHONY: unit-test
unit-test:
	go test -race -v -count=1 -failfast `go list ./... | grep -v client`


## --------------------------------------
## E2E Testing
## --------------------------------------
e2e-install-prerequisites:
	# Download and install kind
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 --output kind && chmod +x kind && sudo mv kind /usr/local/bin/
	# Download and install kubectl
	curl -LO https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl && chmod +x ./kubectl && sudo mv kubectl /usr/local/bin/
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

e2e-kmsv2-setup-kind: setup-local-registry
	./scripts/setup-kmsv2-kind-cluster.sh &
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

e2e-kmsv2-test:
	# Run test suite with kind cluster
	bats -t tests/e2e/testkmsv2.bats
