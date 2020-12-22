binary := kubernetes-kms
REGISTRY_NAME ?= upstreamk8sci
REGISTRY ?= $(REGISTRY_NAME).azurecr.io
REPO_PREFIX ?= k8s/kms
DOCKER_IMAGE := $(REGISTRY)/$(REPO_PREFIX)/keyvault
VERSION ?= v0.0.9
CGO_ENABLED_FLAG := 0

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
	$Q GOOS=${GOOS_FLAG} CGO_ENABLED=${CGO_ENABLED_FLAG} go build .

build-image: authors deps
	@echo "Building..."
	$Q GOOS=linux CGO_ENABLED=${CGO_ENABLED_FLAG} go build .
	@echo "Building docker image..."
	$Q docker build -t $(DOCKER_IMAGE):$(VERSION) .

push: build-image
	$Q docker push $(DOCKER_IMAGE):$(VERSION)

.PHONY: clean deps unit-test integration-test

clean:
	@echo "Clean..."
	$Q rm -rf $(binary)

setup: clean
	@echo "Setup..."
	go get -u github.com/golang/dep/cmd/dep

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
ifndef CI
	@echo "Running Unit Tests outside CI..."
	$Q go env
	go test -race -v -count=1 `go list ./... | grep -v client`
else
	@echo "Running Unit Tests inside CI..."
	go test -race $(shell go list ./... | grep -v /test/e2e) -v
endif

.PHONY: mod
mod:
	@go mod tidy
