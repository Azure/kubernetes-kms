#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

create_kind_cluster () {
  ls -latr
  . scripts/kind-cluster.sh
}

connect_registry () {
  if [ "${kind_network}" != "bridge" ]; then
    # wait for the kind network to exist
    for i in $(seq 1 25); do
      if docker network ls | grep "${kind_network}"; then
        break
      else
        sleep 1
      fi
    done
    containers=$(docker network inspect "${kind_network}" -f "{{range .Containers}}{{.Name}} {{end}}")
    needs_connect="true"
    for c in $containers; do
      if [ "$c" = "${reg_name}" ]; then
        needs_connect="false"
      fi
    done
    if [ "${needs_connect}" = "true" ]; then
      docker network connect "${kind_network}" "${reg_name}" || true
    fi
  fi
}

# desired cluster name; default is "kind"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kms}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.19.0}"

if kind get clusters | grep -q ^kms$ ; then
  echo "cluster already exists, moving on"
  exit 0
fi

# create registry container unless it already exists
kind_version=$(kind version)
kind_network='kind'
reg_name='kind-registry'
reg_port='5000'
case "${kind_version}" in
  "kind v0.7."* | "kind v0.6."* | "kind v0.5."*)
    kind_network='bridge'
    ;;
esac

# create registry container unless it already exists
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run \
    -d --restart=always -p "${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

reg_host="${reg_name}"
if [ "${kind_network}" = "bridge" ]; then
    reg_host="$(docker inspect -f '{{.NetworkSettings.IPAddress}}' "${reg_name}")"
fi
echo "Registry Host: ${reg_host}"

# Build and push kms image
export REGISTRY=localhost:${reg_port}
export IMAGE_NAME=keyvault
export IMAGE_VERSION=e2e-$(git rev-parse --short HEAD)
# push build image to local registry
make push-image
# generate kms plugin manifest and azure.json for testing
make e2e-generate-manifests

create_kind_cluster &
# the registry needs to be connected to the network in parallel
# so the image pull from local registry works. KMS plugin needs to
# start for api-server to respond successfully to health check.
connect_registry &
wait
