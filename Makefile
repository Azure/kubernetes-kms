binary := kubernetes-kms
DOCKER_IMAGE := ritazh/k8s-azure-kms

VERSION          := latest

.PHONY: build
build: authors deps
	@echo "Building..."
	$Q GOOS=linux CGO_ENABLED=0 go build . 
	@echo "Building docker image..."
	$Q docker build -t $(DOCKER_IMAGE):$(VERSION) .

.PHONY: clean deps test testint

deps: setup
	@echo "Ensuring Dependencies..."
	$Q go env
	$Q dep ensure

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

testint:
	@echo "Running Integration tests..."
	$Q sudo GOPATH=$(GOPATH) go test -v -count=1 github.com/Azure/kubernetes-kms/tests/client

test:
	@echo "Running Unit Tests..."
ifndef CI
	@echo "Running Unit Tests outside CI..."
	$Q go env
	$Q dep ensure
	go test -v -count=1 `go list ./... | grep -v client`
else
	@echo "Running Unit Tests inside CI..."
	go test -v `go list ./... | grep -v client`
endif