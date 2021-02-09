#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

create_kind_cluster () {
  # create a cluster with the local registry enabled in containerd
  # add encryption config and the kms static pod manifest with custom image
  cat <<EOF | kind create cluster --retain --image kindest/node:"${KUBERNETES_VERSION}" --name "${KIND_CLUSTER_NAME}" --wait 2m --config=-
  kind: Cluster
  apiVersion: kind.x-k8s.io/v1alpha4
  containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
      endpoint = ["http://${reg_host}:${reg_port}"]
  nodes:
  - role: control-plane
    extraMounts:
    - containerPath: /etc/kubernetes/encryption-config.yaml
      hostPath: tests/e2e/encryption-config.yaml
      readOnly: true
      propagation: None
    - containerPath: /etc/kubernetes/manifests/kubernetes-kms.yaml
      hostPath: tests/e2e/generated_manifests/kms.yaml
      readOnly: true
      propagation: None
    - containerPath: /etc/kubernetes/azure.json
      hostPath: tests/e2e/generated_manifests/azure.json
      readOnly: true
      propagation: None
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        apiServer:
          extraArgs:
            encryption-provider-config: "/etc/kubernetes/encryption-config.yaml"
          extraVolumes:
          - name: encryption-config
            hostPath: "/etc/kubernetes/encryption-config.yaml"
            mountPath: "/etc/kubernetes/encryption-config.yaml"
            readOnly: true
            pathType: File
          - name: sock-path
            hostPath: "/opt"
            mountPath: "/opt"
EOF
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
