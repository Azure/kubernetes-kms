#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# create a cluster with the local registry enabled in containerd
# add encryption config and the kms static pod manifest with custom image
cat <<EOF | kind create cluster --retain --image kindest/node:"${KUBERNETES_VERSION}" --name "${KIND_CLUSTER_NAME}" --wait 2m --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REGISTRY_PORT}"]
    endpoint = ["http://${REGISTRY_NAME}:${REGISTRY_PORT}"]
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