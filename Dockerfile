FROM mcr.microsoft.com/oss/go/microsoft/golang:1.26.3-bookworm@sha256:6ea3c258390542b7d13515be11a31ce797dd5150c9d46ae21c47f0f1ce2786cb AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/server/main.go main.go
COPY pkg/ pkg/

ARG TARGETARCH
ARG TARGETPLATFORM
ARG LDFLAGS
RUN MS_GO_NOSYSTEMCRYPTO=1 CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -ldflags "${LDFLAGS:--X github.com/Azure/kubernetes-kms/pkg/version.BuildVersion=latest}" -o _output/kubernetes-kms main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=${TARGETPLATFORM:-linux/amd64} mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0-nonroot.20260304@sha256:6725a75fb21b0bef01473a1083939b9bd4d43c660d8d89e98906cbcfe717b548
WORKDIR /
COPY --from=builder /workspace/_output/kubernetes-kms .

ENTRYPOINT [ "/kubernetes-kms" ]
