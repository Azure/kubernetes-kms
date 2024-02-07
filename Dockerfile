FROM mcr.microsoft.com/oss/go/microsoft/golang:1.21-bookworm@sha256:970e5548a6334a823204db449c8a65678b1f7757104fcf0365a3bd7489b4a321 as builder

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -ldflags "${LDFLAGS:--X github.com/Azure/kubernetes-kms/pkg/version.BuildVersion=latest}" -o _output/kubernetes-kms main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=${TARGETPLATFORM:-linux/amd64} mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0-nonroot@sha256:5ce81d54a02b6b7378c4cb6c7e45f0ad8c863836c11649536cf7874d07cc3309
WORKDIR /
COPY --from=builder /workspace/_output/kubernetes-kms .

ENTRYPOINT [ "/kubernetes-kms" ]
