REGISTRY_NAME ?= upstreamk8sci
REPO_PREFIX ?= oss/azure/kms
REGISTRY ?= $(REGISTRY_NAME).azurecr.io/$(REPO_PREFIX)
IMAGE_NAME ?= keyvault
IMAGE_VERSION ?= v0.0.10
IMAGE_TAG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
CGO_ENABLED_FLAG := 0

# docker env var
DOCKER_BUILDKIT = 1
export DOCKER_BUILDKIT

ifeq ($(OS),Windows_NT)
	GOOS_FLAG = windows
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S), Linux)
		GOOS_FLAG = linux
	endif
	ifeq ($(UNAME_S), Darwin)
		GOOS_FLAG = darwin
	endif
endif

.PHONY: build
build: authors
	@echo "Building..."
	$Q GOOS=$(GOOS_FLAG) CGO_ENABLED=$(CGO_ENABLED_FLAG) go build -o _output/kubernetes-kms .

build-image: authors clean build
	@echo "Building docker image..."
	$Q docker build -t $(IMAGE_TAG) .

push-image: build-image
	$Q docker push $(IMAGE_TAG)

.PHONY: clean unit-test integration-test

clean:
	@echo "Clean..."
	$Q rm -rf _output/

authors:
	$Q git log --all --format='%aN <%cE>' | sort -u  | sed -n '/github/!p' > GITAUTHORS
	$Q cat AUTHORS GITAUTHORS  | sort -u > NEWAUTHORS
	$Q mv NEWAUTHORS AUTHORS
	$Q rm -f NEWAUTHORS
	$Q rm -f GITAUTHORS

integration-test:
	@echo "Running Integration tests..."
	$Q sudo GOPATH=$(GOPATH) go test -v -count=1 github.com/Azure/kubernetes-kms/tests/client

unit-test:
	@echo "Running Unit Tests..."
	go test -race -v -count=1 `go list ./... | grep -v client`

.PHONY: mod
mod:
	@go mod tidy
