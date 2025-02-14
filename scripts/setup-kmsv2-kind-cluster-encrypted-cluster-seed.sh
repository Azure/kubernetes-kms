#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export ENCRYPTION_CONFIG_FILE=kmsv2-encryption-config-encrypted-cluster-seed.yaml
envsubst < ./tests/e2e/kind-config.yaml > ./tests/e2e/generated_manifests/kind-config.yaml

# create a cluster with the local registry enabled in containerd
# add encryption config and the kms static pod manifest with custom image
kind create cluster --retain --image kindest/node:"${KUBERNETES_VERSION}" --name "${KIND_CLUSTER_NAME}" --wait 2m --config=./tests/e2e/generated_manifests/kind-config.yaml
