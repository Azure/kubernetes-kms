#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# create registry container unless it already exists
running="$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  echo "Creating local registry"
  docker run \
    -d --restart=always -p "${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" \
    mirror.gcr.io/registry:2
fi

# Build and push kms image
export REGISTRY=localhost:${REGISTRY_PORT}
export IMAGE_NAME=keyvault
export IMAGE_VERSION=e2e-$(git rev-parse --short HEAD)
export OUTPUT_TYPE=type=docker

# push build image to local registry
echo "Build and push image to local registry"
make docker-init-buildx docker-build
docker push "${REGISTRY}/${IMAGE_NAME}:${IMAGE_VERSION}"

# generate manifest for local
make e2e-generate-manifests
